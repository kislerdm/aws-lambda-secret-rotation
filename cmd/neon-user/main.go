package main

import (
	"context"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	lambda "github.com/kislerdm/neon-dbpassword-rotation-lambda"
	neon "github.com/kislerdm/neon-sdk-go"
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

	var adminSecret lambda.SecretAdmin
	if err := lambda.ExtractSecretObject(v, &adminSecret); err != nil {
		log.Fatalln(err)
	}

	clientNeon, err := neon.NewClient(neon.WithAPIKey(adminSecret.Token))
	if err != nil {
		log.Fatalf("unable to init Neon SDK, %v", err)
	}

	var s lambda.SecretUser
	lambda.Start(
		lambda.Config{
			SecretsmanagerClient: clientSecretsManager,
			DBClient:             lambda.NewDBClient(clientNeon),
			SecretObj:            &s,
		},
	)
}
