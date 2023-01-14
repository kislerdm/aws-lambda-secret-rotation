package main

import (
	"context"
	"log"

	_ "github.com/lib/pq"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	neon "github.com/kislerdm/neon-sdk-go"
)

type Secret struct {
	User         string `json:"user"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	ProjectID    string `json:"project_id"`
	BranchID     string `json:"branch_id"`
	DatabaseName string `json:"dbname"`
}

func main() {
	cfgSecretsManager, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	clientSecretsManager := secretsmanager.NewFromConfig(cfgSecretsManager)

	clientNeon, err := neon.NewClient()
	if err != nil {
		log.Fatalf("unable to init Neon SDK, %v", err)
	}

	var s Secret
	Start(
		Config{
			SecretsmanagerClient: clientSecretsManager,
			DBClient:             clientDB{c: clientNeon},
			SecretObj:            &s,
		},
	)
}
