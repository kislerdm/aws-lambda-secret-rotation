package confluent

import (
	"context"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
)

const (
	mockIDNew     = "bar-new"
	mockSecretNew = "quxx-456"
)

type mockAPIKeysIamV2Api struct {
	generateCorruptID     bool
	generateCorruptSpec   bool
	generateCorruptSecret bool
	createKeyExecuteError bool
	keys                  map[string]sdk.IamV2ApiKey
}

func (m *mockAPIKeysIamV2Api) CreateIamV2ApiKey(ctx context.Context) sdk.ApiCreateIamV2ApiKeyRequest {
	o := sdk.IamV2ApiKey{
		Id: optString(mockIDNew),
		Spec: &sdk.IamV2ApiKeySpec{
			Secret: optString(mockSecretNew),
		},
	}
	if m.generateCorruptID {
		o.Id = nil
	}
	if m.generateCorruptSecret {
		o.Spec = &sdk.IamV2ApiKeySpec{
			Secret: nil,
		}
	}
	if m.generateCorruptSpec {
		o.Spec = nil
	}
	return sdk.ApiCreateIamV2ApiKeyRequest{
		ApiService: &mockAPIKeysIamV2Api{
			createKeyExecuteError: m.createKeyExecuteError,
			keys:                  map[string]sdk.IamV2ApiKey{mockIDNew: o},
		},
	}
}

func (m *mockAPIKeysIamV2Api) CreateIamV2ApiKeyExecute(r sdk.ApiCreateIamV2ApiKeyRequest) (
	sdk.IamV2ApiKey, *http.Response, error,
) {
	if m.createKeyExecuteError {
		return sdk.IamV2ApiKey{}, nil, errors.New("test-error")
	}
	return m.keys[mockIDNew], nil, nil
}

func (m *mockAPIKeysIamV2Api) DeleteIamV2ApiKey(ctx context.Context, id string) sdk.ApiDeleteIamV2ApiKeyRequest {
	panic("not implemented")
}

func (m *mockAPIKeysIamV2Api) DeleteIamV2ApiKeyExecute(r sdk.ApiDeleteIamV2ApiKeyRequest) (*http.Response, error) {
	panic("not implemented")
}

func (m *mockAPIKeysIamV2Api) GetIamV2ApiKey(ctx context.Context, id string) sdk.ApiGetIamV2ApiKeyRequest {
	v, ok := m.keys[id]
	if !ok {
		return sdk.ApiGetIamV2ApiKeyRequest{
			ApiService: &mockAPIKeysIamV2Api{keys: map[string]sdk.IamV2ApiKey{}},
		}
	}
	return sdk.ApiGetIamV2ApiKeyRequest{
		ApiService: &mockAPIKeysIamV2Api{keys: map[string]sdk.IamV2ApiKey{id: v}},
	}
}

func (m *mockAPIKeysIamV2Api) GetIamV2ApiKeyExecute(r sdk.ApiGetIamV2ApiKeyRequest) (
	sdk.IamV2ApiKey, *http.Response, error,
) {
	for _, v := range m.keys {
		return v, nil, nil
	}
	return sdk.IamV2ApiKey{}, nil, errors.New("not found")
}

func (m *mockAPIKeysIamV2Api) ListIamV2ApiKeys(ctx context.Context) sdk.ApiListIamV2ApiKeysRequest {
	panic("not implemented")
}

func (m *mockAPIKeysIamV2Api) ListIamV2ApiKeysExecute(r sdk.ApiListIamV2ApiKeysRequest) (
	sdk.IamV2ApiKeyList, *http.Response, error,
) {
	panic("not implemented")
}

func (m *mockAPIKeysIamV2Api) UpdateIamV2ApiKey(ctx context.Context, id string) sdk.ApiUpdateIamV2ApiKeyRequest {
	panic("not implemented")
}

func (m *mockAPIKeysIamV2Api) UpdateIamV2ApiKeyExecute(r sdk.ApiUpdateIamV2ApiKeyRequest) (
	sdk.IamV2ApiKey, *http.Response, error,
) {
	panic("not implemented")
}

func Test_dbClient_Create(t *testing.T) {
	const (
		mockSecret      = "qux-123"
		mockDisplayName = "baz"
	)

	type fields struct {
		KeyUser     string
		KeyPassword string
		c           *sdk.APIClient
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
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						keys: map[string]sdk.IamV2ApiKey{
							"bar": {
								Id: optString("bar"),
								Spec: &sdk.IamV2ApiKeySpec{
									Secret:      optString(mockSecret),
									DisplayName: optString(mockDisplayName),
								},
							},
						},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar", "password": mockSecret},
			},
			wantErr: false,
		},
		{
			name: "unhappy path: ID of new key is corrupt",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						generateCorruptID: true,
						keys: map[string]sdk.IamV2ApiKey{
							"bar": {
								Id: optString("bar"),
								Spec: &sdk.IamV2ApiKeySpec{
									Secret:      optString(mockSecret),
									DisplayName: optString(mockDisplayName),
								},
							},
						},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar", "password": mockSecret},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: Spec of new key is corrupt",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						generateCorruptSpec: true,
						keys: map[string]sdk.IamV2ApiKey{
							"bar": {
								Id: optString("bar"),
								Spec: &sdk.IamV2ApiKeySpec{
									Secret:      optString(mockSecret),
									DisplayName: optString(mockDisplayName),
								},
							},
						},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar", "password": mockSecret},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: secret of new key is corrupt",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						generateCorruptSecret: true,
						keys: map[string]sdk.IamV2ApiKey{
							"bar": {
								Id: optString("bar"),
								Spec: &sdk.IamV2ApiKeySpec{
									Secret:      optString(mockSecret),
									DisplayName: optString(mockDisplayName),
								},
							},
						},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar", "password": mockSecret},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: Execute() failed",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						createKeyExecuteError: true,
						keys: map[string]sdk.IamV2ApiKey{
							"bar": {
								Id: optString("bar"),
								Spec: &sdk.IamV2ApiKeySpec{
									Secret:      optString(mockSecret),
									DisplayName: optString(mockDisplayName),
								},
							},
						},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar", "password": mockSecret},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: wrong secret type",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c:           &sdk.APIClient{},
			},
			args: args{
				ctx:    context.TODO(),
				secret: "foobar",
			},
			wantErr: true,
		},
		{
			name: "unhappy path: secret is missing required field",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c:           &sdk.APIClient{},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"foo": "bar"},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: secret with the ID does not exist",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						keys: map[string]sdk.IamV2ApiKey{},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar"},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: secret with the ID exist, but corrupt",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						keys: map[string]sdk.IamV2ApiKey{
							"bar": {},
						},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar"},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: new key is corrupt: no ID",
			fields: fields{
				KeyUser:     "user",
				KeyPassword: "password",
				c: &sdk.APIClient{
					APIKeysIamV2Api: &mockAPIKeysIamV2Api{
						keys: map[string]sdk.IamV2ApiKey{
							"bar": {},
						},
					},
				},
			},
			args: args{
				ctx:    context.TODO(),
				secret: SecretUser{"user": "bar"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				c := dbClient{
					KeyUser:     tt.fields.KeyUser,
					KeyPassword: tt.fields.KeyPassword,
					c:           tt.fields.c,
				}
				if err := c.Create(tt.args.ctx, tt.args.secret); (err != nil) != tt.wantErr {
					t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				}

				if !tt.wantErr {
					s := tt.args.secret.(SecretUser)
					if mockIDNew != s["user"] || mockSecretNew != s["password"] {
						t.Errorf("Create() newly generated secret was not stored correctly")
					}
				}
			},
		)
	}
}

func optString(s string) *string {
	return &s
}
