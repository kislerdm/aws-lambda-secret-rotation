package main

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsmanagerTriggerPayload defines the AWS Lambda function event payload type.
type SecretsmanagerTriggerPayload struct {
	// The secret ARN or identifier
	SecretARN string `json:"SecretId"`
	// The ClientRequestToken of the secret version
	Token string `json:"ClientRequestToken"`
	// The rotation step (one of createSecret, setSecret, testSecret, or finishSecret)
	Step string `json:"Step"`
}

// DBClient defines the interface to handle database communication to rotate the access credentials.
type DBClient interface {
	// SetSecret sets the password to a user in the database.
	SetSecret(secret interface{}) error

	// TestSecret tests the database access using the secret from the stage AWSPENDING.
	TestSecret(secret interface{}) error
}

type lambdaHandler func(ctx context.Context, event SecretsmanagerTriggerPayload) error

type secretsManager struct {
	c *secretsmanager.Client
}

func (s secretsManager) CreateSecret(ctx context.Context, event SecretsmanagerTriggerPayload) error {
	var (
		v   *secretsmanager.GetSecretValueOutput
		err error
	)
	v, err = s.c.GetSecretValue(
		ctx, &secretsmanager.GetSecretValueInput{
			SecretId:     aws.String(event.SecretARN),
			VersionId:    aws.String(event.Token),
			VersionStage: aws.String("AWSCURRENT"),
		},
	)
	if err != nil {
		v, err = s.c.GetSecretValue(
			ctx, &secretsmanager.GetSecretValueInput{
				SecretId: aws.String(event.SecretARN),
			},
		)
		if err != nil {
			return err
		}
		err = nil
	}

	return nil
}

func (s secretsManager) SetSecret(ctx context.Context, event SecretsmanagerTriggerPayload, dbHandler DBClient) error {
	//TODO implement me
	panic("implement me")
}

func (s secretsManager) TestSecret(ctx context.Context, event SecretsmanagerTriggerPayload, dbHandler DBClient) error {
	//TODO implement me
	panic("implement me")
}

func (s secretsManager) FinishSecret(ctx context.Context, event SecretsmanagerTriggerPayload) error {
	//TODO implement me
	panic("implement me")
}

func router(dbClient DBClient, sm secretsManagerClient) lambdaHandler {
	return func(ctx context.Context, event SecretsmanagerTriggerPayload) error {
		switch s := event.Step; s {
		case "createSecret":
			return sm.CreateSecret(ctx, event)
		case "setSecret":
			return sm.SetSecret(ctx, event, dbClient)
		case "testSecret":
			return sm.TestSecret(ctx, event, dbClient)
		case "finishSecret":
			return sm.FinishSecret(ctx, event)
		default:
			return errors.New("unknown step " + s)
		}
	}
}

// Start proxy to lambda lambdaHandler which handles inter.
func Start(dbClient DBClient, secretsManagerClient *secretsmanager.Client) {
	lambda.Start(router(dbClient, secretsManager{c: secretsManagerClient}))
}

type secretsManagerClient interface {
	// CreateSecret the method first checks for the existence of a secret for the passed in secretARN.
	// If one does not exist, it will generate a new secret and put it with the passed in secretARN.
	CreateSecret(ctx context.Context, event SecretsmanagerTriggerPayload) error

	// SetSecret sets the AWSPENDING secret in the service that the secret belongs to.
	// For example, if the secret is a database credential,
	// this method should take the value of the AWSPENDING secret
	// and set the user's password to this value in the database.
	SetSecret(ctx context.Context, event SecretsmanagerTriggerPayload, dbHandler DBClient) error

	// TestSecret the method tries to log into the database with the secrets staged with AWSPENDING.
	TestSecret(ctx context.Context, event SecretsmanagerTriggerPayload, dbHandler DBClient) error

	// FinishSecret the method finishes the secret rotation
	// by setting the secret staged AWSPENDING with the AWSCURRENT stage.
	FinishSecret(ctx context.Context, event SecretsmanagerTriggerPayload) error
}
