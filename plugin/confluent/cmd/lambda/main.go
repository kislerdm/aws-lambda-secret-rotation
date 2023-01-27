package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	confluentClient "github.com/kislerdm/aws-lambda-secret-rotation/plugin/confluent-kafka"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sdk "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	secretRotation "github.com/kislerdm/aws-lambda-secret-rotation"
)

// userAgent mimics terraform UserAgent.
func userAgent() string {
	// terraform sdk version
	// https://pkg.go.dev/github.com/hashicorp/terraform-plugin-sdk/v2/meta@v2.24.1#SDKVersion
	const (
		terraformSDKVersion = "2.10.1"
		terraformVersion    = "v1.3.3"
	)
	return "Terraform/" + terraformVersion + " (+https://www.terraform.io) Terraform-Plugin-SDK/" + terraformSDKVersion
}

func main() {
	secretAdminARN := os.Getenv("ADMIN_SECRET_ARN")
	if secretAdminARN == "" {
		log.Fatalln("ADMIN_SECRET_ARN env. variable must be set")
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

	var adminSecret confluentClient.SecretAdmin
	if err := secretRotation.ExtractSecretObject(v, &adminSecret); err != nil {
		log.Fatalln(err)
	}

	cfg := sdk.NewConfiguration()
	cfg.Servers[0].URL = "https://api.confluent.cloud"
	cfg.UserAgent = userAgent()

	client := sdk.NewAPIClient(cfg)

	var s confluentClient.SecretUser
	handler, err := secretRotation.NewHandler(
		secretRotation.Config{
			SecretsmanagerClient: clientSecretsManager,
			ServiceClient: confluentClient.NewServiceClient(
				client, os.Getenv("ATTRIBUTE_KEY"), os.Getenv("ATTRIBUTE_SECRET"),
			),
			SecretObj: &s,
			Debug:     secretRotation.StrToBool(os.Getenv("DEBUG")),
		},
	)
	if err != nil {
		log.Fatalf("unable to init lambda handler to rotate secret, %v", err)
	}

	lambda.Start(handler)
}
