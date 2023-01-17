package lambda

import (
	"context"
	"database/sql"
	"errors"

	neon "github.com/kislerdm/neon-sdk-go"
)

// NewDBClient initiates the `DBClient` to rotate credentials for Neon user.
func NewDBClient(client neon.Client) DBClient {
	return &dbClient{c: client}
}

type dbClient struct {
	c neon.Client
}

func (c dbClient) SetSecret(ctx context.Context, secretCurrent, secretPending, secretPrevious any) error {
	return nil
}

func (c dbClient) TryConnection(ctx context.Context, secret any) error {
	db, err := c.openDBConnection(secret)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	return db.PingContext(ctx)
}

func (c dbClient) GenerateSecret(ctx context.Context, secret any) error {
	s, ok := secret.(*SecretUser)
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

func (c dbClient) openDBConnection(secret any) (db, error) {
	s, ok := secret.(*SecretUser)
	if !ok {
		return nil, errors.New("wrong secret type")
	}

	if s.User == "" || s.DatabaseName == "" || s.Host == "" {
		return nil, errors.New("failed to connect")
	}

	connStr := "user=" + s.User +
		" dbname=" + s.DatabaseName +
		" host=" + s.Host +
		" sslmode=verify-full"

	if s.Password != "" {
		connStr += " password=" + s.Password
	}

	if s.Host == "dev" {
		if s.DatabaseName == "fail" {
			return mockDB{FailedPing: true}, nil
		}
		return mockDB{}, nil
	}

	return sql.Open("postgres", connStr)
}
