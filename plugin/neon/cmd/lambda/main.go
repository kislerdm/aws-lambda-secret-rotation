package main

import (
	"context"
	"log"
	"os"

	dbclient "github.com/kislerdm/password-rotation-lambda/plugin/neon"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sdk "github.com/kislerdm/neon-sdk-go"
	lambda "github.com/kislerdm/password-rotation-lambda"
)

func main() {
	secretAdminARN := os.Getenv("NEON_TOKEN_SECRET_ARN")
	if secretAdminARN == "" {
		log.Fatalln("NEON_TOKEN_SECRET_ARN env. variable must be set")
	}

	cfgSecretsManager, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	clientSecretsManager := secretsmanager.NewFromConfig(cfgSecretsManager)

	v, err := clientSecretsManager.GetSecretValue(
		context.Background(), &secretsmanager.GetSecretValueInput{SecretId: &secretAdminARN},
	)
	if err != nil {
		log.Fatalln(err)
	}

	var adminSecret dbclient.SecretAdmin
	if err := lambda.ExtractSecretObject(v, &adminSecret); err != nil {
		log.Fatalln(err)
	}

	clientNeon, err := sdk.NewClient(sdk.WithAPIKey(adminSecret.Token))
	if err != nil {
		log.Fatalf("unable to init Neon SDK, %v", err)
	}

	var s dbclient.SecretUser
	lambda.Start(
		lambda.Config{
			SecretsmanagerClient: clientSecretsManager,
			ServiceClient:        dbclient.NewServiceClient(clientNeon),
			SecretObj:            &s,
			Debug:                lambda.StrToBool(os.Getenv("DEBUG")),
		},
	)
}
