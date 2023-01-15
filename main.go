package main

import (
	"context"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	neon "github.com/kislerdm/neon-sdk-go"
)

// SecretAdmin defines the secret with the db admin access details.
type SecretAdmin struct {
	Token string `json:"token"`
}

// SecretUser defines the secret with db user access details.
type SecretUser struct {
	User         string `json:"user"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	ProjectID    string `json:"project_id"`
	BranchID     string `json:"branch_id"`
	DatabaseName string `json:"dbname"`
}

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

	var adminSecret SecretAdmin
	if err := extractSecretObject(v, &adminSecret); err != nil {
		log.Fatalln(err)
	}

	clientNeon, err := neon.NewClient(neon.WithAPIKey(adminSecret.Token))
	if err != nil {
		log.Fatalf("unable to init Neon SDK, %v", err)
	}

	var s SecretUser
	Start(
		Config{
			SecretsmanagerClient: clientSecretsManager,
			DBClient:             clientDB{c: clientNeon},
			SecretObj:            &s,
		},
	)
}
