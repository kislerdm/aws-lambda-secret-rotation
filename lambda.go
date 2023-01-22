package lambda

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"unsafe"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"
	smithyHttp "github.com/aws/smithy-go/transport/http"
)

// Config defines the rotation lambda's configuration.
type Config struct {
	// SecretsmanagerClient the client's instance to communicate with the secretsmanager.
	SecretsmanagerClient SecretsmanagerClient

	// ServiceClient the client's instance to communicate with the service delegated credentials storage.
	ServiceClient ServiceClient

	// SecretObj defines the interface of the secret to rotate.
	SecretObj any

	// Debug set to `true` to activate debug level logs.
	Debug bool
}

// secretsmanagerTriggerPayload defines the AWS Lambda function's event payload type.
type secretsmanagerTriggerPayload struct {
	// The secret ARN or identifier
	SecretARN string `json:"SecretId"`

	// The ClientRequestToken of the secret version
	Token string `json:"ClientRequestToken"`

	// The rotation step (one of createSecret, setSecret, testSecret, or finishSecret)
	Step string `json:"Step"`
}

// Start defines the lambda handler.
func Start(cfg Config) {
	lambda.Start(
		func(ctx context.Context, event secretsmanagerTriggerPayload) error {
			if cfg.Debug {
				log.Println(
					"[DEBUG] arn: " + event.SecretARN + "; step: " + event.Step + "; token: " + event.Token + "\n",
				)
			}
			if err := validateEvent(ctx, event, cfg.SecretsmanagerClient); err != nil {
				if cfg.Debug {
					log.Println("[DEBUG] validation error:+" + err.Error() + "\n")
				}
				return err
			}

			// routes to appropriate step.
			switch s := event.Step; s {
			case "createSecret":
				return createSecret(ctx, event, cfg)
			case "setSecret":
				return setSecret(ctx, event, cfg)
			case "testSecret":
				return testSecret(ctx, event, cfg)
			case "finishSecret":
				return finishSecret(ctx, event, cfg)
			default:
				return errors.New("unknown step " + s)
			}
		},
	)
}

// SecretsmanagerClient client to communicate with the secretsmanager.
type SecretsmanagerClient interface {
	GetSecretValue(
		ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.GetSecretValueOutput, error)

	PutSecretValue(
		ctx context.Context, input *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.PutSecretValueOutput, error)

	DescribeSecret(
		ctx context.Context, input *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options),
	) (
		*secretsmanager.DescribeSecretOutput, error,
	)

	UpdateSecretVersionStage(
		ctx context.Context, input *secretsmanager.UpdateSecretVersionStageInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.UpdateSecretVersionStageOutput, error)
}

// ServiceClient defines the interface to communicate with the service (e.g. database) to rotate the access credentials.
type ServiceClient interface {
	// Create generates the secret and mutates the `secret` value.
	Create(ctx context.Context, secret any) error

	// Set sets newly generated credentials in the system delegated credentials storage.
	Set(ctx context.Context, secretCurrent, secretPending, secretPrevious any) error

	// Test tries to connect to the system delegated credentials storage using newly generated secret.
	Test(ctx context.Context, secret any) error
}

// validateEvent checks if the secret version is staged correctly.
func validateEvent(ctx context.Context, event secretsmanagerTriggerPayload, client SecretsmanagerClient) error {
	v, err := client.DescribeSecret(
		ctx, &secretsmanager.DescribeSecretInput{
			SecretId: aws.String(event.SecretARN),
		},
	)
	if err != nil {
		return err
	}

	if v.RotationEnabled == nil || !aws.ToBool(v.RotationEnabled) {
		return errors.New("secret " + event.SecretARN + " is not enabled for rotation")
	}

	versions, ok := v.VersionIdsToStages[event.Token]
	if !ok || len(versions) == 0 {
		return errors.New("secret version " + event.Token + " has no stage for rotation of secret " + event.SecretARN)
	}

	return nil
}

// createSecret the method first checks for the existence of a secret for the passed in secretARN.
// If one does not exist, it will generate a new secret and put it with the passed in secretARN.
func createSecret(ctx context.Context, event secretsmanagerTriggerPayload, cfg Config) error {
	if cfg.Debug {
		log.Println("[DEBUG] Fetch AWSCURRENT of the secret: " + event.SecretARN)
	}
	v, err := getSecretValue(ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSCURRENT", "")
	if err != nil {
		if cfg.Debug {
			if cfg.Debug {
				log.Println("[DEBUG] error: " + err.Error())
			}
		}
		return err
	}

	if cfg.Debug {
		log.Println(
			"[DEBUG] Check if stage AWSPENDING exists for the version: " + event.Token + " of the secret: " +
				event.SecretARN,
		)
	}
	if _, err := getSecretValue(
		ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPENDING", event.Token,
	); nil == err {
		if cfg.Debug {
			log.Println("[DEBUG] AWSPENDING exists, return.")
		}
		return nil
	}

	if cfg.Debug {
		log.Println("[DEBUG] Deserialize secret from the stage AWSCURRENT")
	}
	if err := ExtractSecretObject(v, cfg.SecretObj); err != nil {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] Generate new secret")
	}
	if err := cfg.ServiceClient.Create(ctx, cfg.SecretObj); err != nil {
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] Serialize newly generated secret")
	}
	o, err := serialiseSecret(cfg.SecretObj)
	if err != nil {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] Put newly generated secret to AWSPENDING stage")
	}
	_, err = cfg.SecretsmanagerClient.PutSecretValue(
		ctx, &secretsmanager.PutSecretValueInput{
			SecretId:           aws.String(event.SecretARN),
			ClientRequestToken: aws.String(event.Token),
			SecretString:       o,
			VersionStages:      []string{"AWSPENDING"},
		},
	)
	if err != nil && cfg.Debug {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
	}
	return err
}

// setSecret sets the AWSPENDING secret in the service that the secret belongs to.
// For example, if the secret is a database credential,
// this method should take the value of the AWSPENDING secret
// and set the user's password to this value in the database.
func setSecret(ctx context.Context, event secretsmanagerTriggerPayload, cfg Config) error {
	if cfg.Debug {
		log.Println("[DEBUG] Fetch AWSPREVIOUS of the secret: " + event.SecretARN)
	}
	secretPrevious, err := getSecretValue(ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPREVIOUS", "")
	switch err.(type) {
	case *types.ResourceNotFoundException, nil:
	case *smithy.OperationError:
		if e, ok := err.(*smithy.OperationError).Unwrap().(*smithyHttp.ResponseError); ok {
			switch e.HTTPStatusCode() {
			case http.StatusBadRequest, http.StatusNotFound:
			default:
				return err
			}
		}
	default:
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] Fetch AWSCURRENT of the secret: " + event.SecretARN)
	}
	secretCurrent, err := getSecretValue(ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSCURRENT", "")
	if err != nil {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] Fetch AWSPENDING of the secret: " + event.SecretARN)
	}
	secretPending, err := getSecretValue(
		ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPENDING", event.Token,
	)
	if err != nil {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] call cfg.ServiceClient.Set()")
	}
	return cfg.ServiceClient.Set(ctx, secretCurrent, secretPending, secretPrevious)
}

// testSecret the method tries to log into the database with the secrets staged with AWSPENDING.
func testSecret(ctx context.Context, event secretsmanagerTriggerPayload, cfg Config) error {
	if cfg.Debug {
		log.Println("[DEBUG] Fetch AWSPENDING of the secret: " + event.SecretARN + ", version: " + event.Token)
	}
	v, err := getSecretValue(
		ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPENDING", event.Token,
	)
	if err != nil {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] deserialize secret value")
	}
	if err := ExtractSecretObject(v, cfg.SecretObj); err != nil {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	if cfg.Debug {
		log.Println("[DEBUG] try to connect to database")
	}
	return cfg.ServiceClient.Test(ctx, cfg.SecretObj)
}

// finishSecret the method finishes the secret rotation
// by setting the secret staged AWSPENDING with the AWSCURRENT stage.
func finishSecret(ctx context.Context, event secretsmanagerTriggerPayload, cfg Config) error {
	if cfg.Debug {
		log.Println("[DEBUG] Describe secret: " + event.SecretARN)
	}
	v, err := cfg.SecretsmanagerClient.DescribeSecret(
		ctx, &secretsmanager.DescribeSecretInput{
			SecretId: aws.String(event.SecretARN),
		},
	)
	if err != nil {
		if cfg.Debug {
			log.Println("[DEBUG] error: " + err.Error())
		}
		return err
	}

	currentVersion := ""
	if vv := v.VersionIdsToStages; vv != nil {
		for version, stages := range vv {
			for _, stage := range stages {
				if "AWSCURRENT" == stage {
					if event.Token == version {
						if cfg.Debug {
							log.Println("[DEBUG] version " + version + " is already at the stage AWSCURRENT")
						}
						return nil
					}
					currentVersion = version
				}
			}
		}
	}

	if cfg.Debug {
		log.Println("[DEBUG] update version from " + currentVersion + " to AWSCURRENT")
	}
	_, err = cfg.SecretsmanagerClient.UpdateSecretVersionStage(
		ctx, &secretsmanager.UpdateSecretVersionStageInput{
			SecretId:            aws.String(event.SecretARN),
			VersionStage:        aws.String("AWSCURRENT"),
			MoveToVersionId:     aws.String(event.Token),
			RemoveFromVersionId: aws.String(currentVersion),
		},
	)
	return err
}

// StrToBool converts string to bool.
func StrToBool(s string) bool {
	switch s = strings.ToLower(s); s {
	case "yes", "true", "1":
		return true
	default:
		return false
	}
}

// ExtractSecretObject deserializes secret value to a Go object of the secret type.
func ExtractSecretObject(v *secretsmanager.GetSecretValueOutput, secret any) error {
	return json.Unmarshal([]byte(*v.SecretString), secret)
}

func serialiseSecret(secret any) (*string, error) {
	o, err := json.Marshal(secret)
	if err != nil {
		return nil, err
	}
	return (*string)(unsafe.Pointer(&o)), nil
}

func getSecretValue(
	ctx context.Context, client SecretsmanagerClient, secretARN, stage, version string,
) (*secretsmanager.GetSecretValueOutput, error) {
	params := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretARN),
		VersionStage: aws.String(stage),
	}
	if version != "" {
		params.VersionId = aws.String(version)
	}
	return client.GetSecretValue(ctx, params)
}
