package main

import (
	"context"
	"encoding/json"
	"errors"
	"unsafe"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

func extractSecretObject(v *secretsmanager.GetSecretValueOutput, secret any) error {
	return json.Unmarshal([]byte(*v.SecretString), secret)
}

func serialiseSecret(secret any) (*string, error) {
	o, err := json.Marshal(secret)
	if err != nil {
		return nil, err
	}
	return (*string)(unsafe.Pointer(&o)), nil
}

// createSecret the method first checks for the existence of a secret for the passed in secretARN.
// If one does not exist, it will generate a new secret and put it with the passed in secretARN.
func createSecret(ctx context.Context, event SecretsmanagerTriggerPayload, cfg Config) error {
	v, err := getSecretValue(ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSCURRENT", "")
	if err != nil {
		return err
	}

	if _, err := getSecretValue(
		ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPENDING", event.Token,
	); nil == err {
		return nil
	}

	if err := extractSecretObject(v, cfg.SecretObj); err != nil {
		return err
	}

	if err := cfg.DBClient.GenerateSecret(ctx, cfg.SecretObj); err != nil {
		return err
	}

	o, err := serialiseSecret(cfg.SecretObj)
	if err != nil {
		return err
	}

	_, err = cfg.SecretsmanagerClient.PutSecretValue(
		ctx, &secretsmanager.PutSecretValueInput{
			SecretId:           aws.String(event.SecretARN),
			ClientRequestToken: aws.String(event.Token),
			SecretString:       o,
			VersionStages:      []string{"AWSPENDING"},
		},
	)
	return err
}

func getSecretValue(
	ctx context.Context, client SecretsmanagerClient, secretARN, stage, version string,
) (*secretsmanager.GetSecretValueOutput, error) {
	return client.GetSecretValue(
		ctx, &secretsmanager.GetSecretValueInput{
			SecretId:     aws.String(secretARN),
			VersionStage: aws.String(stage),
			VersionId:    aws.String(version),
		},
	)
}

// setSecret sets the AWSPENDING secret in the service that the secret belongs to.
// For example, if the secret is a database credential,
// this method should take the value of the AWSPENDING secret
// and set the user's password to this value in the database.
func setSecret(ctx context.Context, event SecretsmanagerTriggerPayload, cfg Config) error {
	secretPrevious, err := getSecretValue(ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPREVIOUS", "")
	switch err.(type) {
	case *types.ResourceNotFoundException, nil:
	default:
		return err
	}

	secretCurrent, err := getSecretValue(ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSCURRENT", "")
	if err != nil {
		return err
	}

	secretPending, err := getSecretValue(
		ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPENDING", event.Token,
	)
	if err != nil {
		return err
	}

	return cfg.DBClient.SetSecret(ctx, secretCurrent, secretPending, secretPrevious)
}

// testSecret the method tries to log into the database with the secrets staged with AWSPENDING.
func testSecret(ctx context.Context, event SecretsmanagerTriggerPayload, cfg Config) error {
	v, err := getSecretValue(
		ctx, cfg.SecretsmanagerClient, event.SecretARN, "AWSPENDING", event.Token,
	)
	if err != nil {
		return err
	}

	return cfg.DBClient.TryConnection(ctx, v)
}

// finishSecret the method finishes the secret rotation
// by setting the secret staged AWSPENDING with the AWSCURRENT stage.
func finishSecret(ctx context.Context, event SecretsmanagerTriggerPayload, cfg Config) error {
	v, err := cfg.SecretsmanagerClient.DescribeSecret(
		ctx, &secretsmanager.DescribeSecretInput{
			SecretId: aws.String(event.SecretARN),
		},
	)
	if err != nil {
		return err
	}

	currentVersion := ""

	if vv := v.VersionIdsToStages; vv != nil {
		for version, stages := range vv {
			for _, stage := range stages {
				if "AWSCURRENT" == stage {
					if event.Token == version {
						return nil
					}
				}
			}
			currentVersion = version
		}
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

// DBClient defines the interface to handle database communication to rotate the access credentials.
type DBClient interface {
	// SetSecret sets the password to a user in the database.
	SetSecret(ctx context.Context, secretCurrent, secretPending, secretPrevious any) error

	// TryConnection tries to connect to the database, and executes a dummy statement.
	TryConnection(ctx context.Context, secret any) error

	// GenerateSecret generates the secret and mutates the `secret` value.
	GenerateSecret(ctx context.Context, secret any) error
}

// SecretsmanagerTriggerPayload defines the AWS Lambda function event payload type.
type SecretsmanagerTriggerPayload struct {
	// The secret ARN or identifier
	SecretARN string `json:"SecretId"`
	// The ClientRequestToken of the secret version
	Token string `json:"ClientRequestToken"`
	// The rotation step (one of createSecret, setSecret, testSecret, or finishSecret)
	Step string `json:"Step"`
}

// Config defines the rotation lambda's configuration.
type Config struct {
	SecretsmanagerClient SecretsmanagerClient
	DBClient             DBClient
	SecretObj            any
}

// Start proxy to lambda lambdaHandler which handles inter.
func Start(cfg Config) {
	lambda.Start(
		func(ctx context.Context, event SecretsmanagerTriggerPayload) error {
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
