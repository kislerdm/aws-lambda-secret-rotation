package main

import (
	"context"
	"database/sql"
	"errors"
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

type clientDB struct {
	c neon.Client
}

func (c clientDB) SetSecret(ctx context.Context, secret any) error {
	return nil
}

func (c clientDB) TryConnection(ctx context.Context, secret any) error {
	db, err := c.openDBConnection(secret)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	return db.PingContext(ctx)
}

func (c clientDB) GenerateSecret(ctx context.Context, secret any) error {
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

type db interface {
	Close() error
	PingContext(ctx context.Context) error
}

type mockDB struct {
	FailedPing bool
}

func (m mockDB) Close() error {
	return nil
}

func (m mockDB) PingContext(ctx context.Context) error {
	if m.FailedPing {
		return errors.New("failed to query")
	}
	return nil
}

func (c clientDB) openDBConnection(secret any) (db, error) {
	s, ok := secret.(Secret)
	if !ok {
		return nil, errors.New("wrong secret type")
	}

	connStr := "user= " + s.User +
		"password=" + s.Password +
		"dbname=" + s.DatabaseName +
		"host=" + s.Host +
		"sslmode=verify-full"

	if s.Host == "dev" {
		return mockDB{}, nil
	}

	if s.Host == "fail" {
		return nil, errors.New("failed to connect")
	}

	if s.DatabaseName == "fail" {
		return mockDB{FailedPing: true}, nil
	}

	return sql.Open("postgres", connStr)
}
