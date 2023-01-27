package confluent

import (
	"context"
	"errors"

	sdk "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	lambda "github.com/kislerdm/aws-lambda-secret-rotation"
)

// NewServiceClient initiates the `ServiceClient` to rotate credentials for Confluent Kafka user.
func NewServiceClient(client *sdk.APIClient) lambda.ServiceClient {
	return &dbClient{c: client}
}

type dbClient struct {
	KeyUser     string
	KeyPassword string
	c           *sdk.APIClient
}

func (c dbClient) Set(ctx context.Context, secretCurrent, secretPending, secretPrevious any) error {
	panic("todo")
}

func (c dbClient) Test(ctx context.Context, secret any) error {
	panic("todo")
}

func (c dbClient) Create(ctx context.Context, secret any) error {
	s, ok := secret.(SecretUser)
	if !ok {
		return errors.New("wrong secret type")
	}

	id, ok := s[c.KeyUser]
	if !ok {
		return errors.New("wrong secret type: 'user' field")
	}

	currentKey, err := readKey(ctx, c.c.APIKeysIamV2Api, id)
	if err != nil {
		return err
	}

	spec := currentKey.GetSpec()
	spec.SetSecret("")
	spec.SetDisplayName(spec.GetDisplayName() + "-rotate")

	createdKey, err := createKey(ctx, c.c.APIKeysIamV2Api, &spec)
	if err != nil {
		return err
	}

	s[c.KeyUser] = createdKey.GetId()

	sp, _ := createdKey.GetSpecOk()
	s[c.KeyPassword] = sp.GetSecret()

	secret = s
	return nil
}

func createKey(ctx context.Context, c sdk.APIKeysIamV2Api, spec *sdk.IamV2ApiKeySpec) (*sdk.IamV2ApiKey, error) {
	r := c.CreateIamV2ApiKey(ctx).IamV2ApiKey(sdk.IamV2ApiKey{Spec: spec})
	key, _, err := r.Execute()
	if err != nil {
		return nil, err
	}
	if _, ok := key.GetIdOk(); !ok {
		return nil, errors.New("new API Key is corrupt: ID is either empty or nil")
	}
	s, ok := key.GetSpecOk()
	if !ok {
		return nil, errors.New("new API Key is corrupt: no spec found")
	}
	if _, ok := s.GetSecretOk(); !ok {
		return nil, errors.New("new API Key is corrupt: Secret is either empty or nil")
	}
	return &key, err
}

func readKey(ctx context.Context, c sdk.APIKeysIamV2Api, id string) (*sdk.IamV2ApiKey, error) {
	r := c.GetIamV2ApiKey(ctx, id)
	key, _, err := r.Execute()
	if err != nil {
		return nil, err
	}
	if _, ok := key.GetIdOk(); !ok {
		return nil, errors.New("existing API Key is corrupt: ID is either empty or nil")
	}
	return &key, err
}
