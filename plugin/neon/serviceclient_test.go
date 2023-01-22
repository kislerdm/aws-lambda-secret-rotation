package neon

import (
	"context"
	"testing"

	sdk "github.com/kislerdm/neon-sdk-go"
)

func newMockSDKClient() sdk.Client {
	c, err := sdk.NewClient(sdk.WithHTTPClient(sdk.NewMockHTTPClient()))
	if err != nil {
		panic(err)
	}
	return c
}

const placeholderPassword = "quxx"

func Test_clientDB_GenerateSecret(t *testing.T) {
	type fields struct {
		c sdk.Client
	}
	type args struct {
		ctx    context.Context
		secret any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				c: newMockSDKClient(),
			},
			args: args{
				ctx: context.TODO(),
				secret: &SecretUser{
					User:      "qux",
					ProjectID: "foo",
					BranchID:  "br-bar",
					Password:  placeholderPassword,
				},
			},
			wantErr: false,
		},
		{
			name: "unhappy path: wrong secret type",
			fields: fields{
				c: newMockSDKClient(),
			},
			args: args{
				ctx: context.TODO(),
				secret: SecretUser{
					User:      "qux",
					ProjectID: "foo",
					BranchID:  "br-bar",
					Password:  placeholderPassword,
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: user not found",
			fields: fields{
				c: newMockSDKClient(),
			},
			args: args{
				ctx: context.TODO(),
				secret: &SecretUser{
					User:      "missing",
					ProjectID: "foo",
					BranchID:  "br-bar",
					Password:  placeholderPassword,
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: missing user",
			fields: fields{
				c: newMockSDKClient(),
			},
			args: args{
				ctx: context.TODO(),
				secret: &SecretUser{
					User:      "",
					ProjectID: "foo",
					BranchID:  "br-bar",
					Password:  placeholderPassword,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				c := dbClient{
					c: tt.fields.c,
				}
				err := c.GenerateSecret(tt.args.ctx, tt.args.secret)
				if (err != nil) != tt.wantErr {
					t.Errorf("GenerateSecret() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && tt.args.secret.(*SecretUser).Password == placeholderPassword {
					t.Errorf("GenerateSecret() failed to mutate a SecretUser obj")
				}
			},
		)
	}
}

func Test_clientDB_TryConnection(t *testing.T) {
	type fields struct {
		c sdk.Client
	}
	type args struct {
		ctx    context.Context
		secret any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				c: newMockSDKClient(),
			},
			args: args{
				ctx: context.TODO(),
				secret: &SecretUser{
					User:         "qux",
					Host:         "dev",
					DatabaseName: "baz",
					ProjectID:    "foo",
					BranchID:     "br-bar",
					Password:     placeholderPassword,
				},
			},
			wantErr: false,
		},
		{
			name: "unhappy path: wrong secret content - missing host",
			fields: fields{
				c: newMockSDKClient(),
			},
			args: args{
				ctx: context.TODO(),
				secret: &SecretUser{
					User:         "qux",
					ProjectID:    "foo",
					DatabaseName: "baz",
					BranchID:     "br-bar",
					Password:     placeholderPassword,
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: failed to ping",
			fields: fields{
				c: newMockSDKClient(),
			},
			args: args{
				ctx: context.TODO(),
				secret: &SecretUser{
					User:         "qux",
					ProjectID:    "foo",
					Host:         "dev",
					DatabaseName: "fail",
					BranchID:     "br-bar",
					Password:     placeholderPassword,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				c := dbClient{
					c: tt.fields.c,
				}
				if err := c.TryConnection(tt.args.ctx, tt.args.secret); (err != nil) != tt.wantErr {
					t.Errorf("TryConnection() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}
