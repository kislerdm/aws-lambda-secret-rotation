package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	neon "github.com/kislerdm/neon-sdk-go"
)

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

}
