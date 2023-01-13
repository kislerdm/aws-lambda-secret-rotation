package main

import (
	"context"
	"errors"
	"log"

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
	DatabaseName string `json:"database_name"`
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

type clientDB struct {
	c neon.Client
}

func (c clientDB) SetSecret(secret interface{}) error {
	return nil
}

func (c clientDB) TryConnection(secret interface{}) error {
	panic("todo")
}

func (c clientDB) GenerateSecret(secret interface{}) error {
	s, ok := secret.(Secret)
	if !ok {
		return errors.New("wrong secret type")
	}

	o, err := c.c.ResetProjectBranchRolePassword(s.ProjectID, s.BranchID, s.User)
	if err != nil {
		return err
	}

	s.Password = o.RoleResponse.Role.Password

	return nil
}
