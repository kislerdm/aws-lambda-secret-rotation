package confluent

import (
	"context"
	"errors"
	"reflect"

	sdk "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	lambda "github.com/kislerdm/aws-lambda-secret-rotation"
)

// NewServiceClient initiates the `ServiceClient` to rotate credentials for Confluent Kafka user.
func NewServiceClient(
	client *sdk.APIClient, apiKey, apiSecret, attributeKey, attributeSecret string,
) (lambda.ServiceClient, error) {
	if apiKey == "" || apiSecret == "" {
		return nil, errors.New("confluent API key-secret pair must be provided")
	}
	if attributeKey == "" {
		attributeKey = "user"
	}
	if attributeSecret == "" {
		attributeSecret = "password"
	}
	return &dbClient{
		c:               client,
		attributeKey:    attributeKey,
		attributeSecret: attributeSecret,
		apiKey:          apiKey,
		apiSecret:       apiSecret,
	}, nil
}

type dbClient struct {
	apiKey          string
	apiSecret       string
	attributeKey    string
	attributeSecret string
	c               *sdk.APIClient
}

func (c dbClient) wrapContext(ctx context.Context) context.Context {
	return context.WithValue(
		ctx, sdk.ContextBasicAuth, sdk.BasicAuth{
			UserName: c.apiKey,
			Password: c.apiSecret,
		},
	)
}

func (c dbClient) Set(ctx context.Context, secretCurrent, secretPending, secretPrevious any) error {
	ctx = c.wrapContext(ctx)

	if err := c.Test(ctx, secretCurrent); err != nil {
		return errors.New("current secret error: " + err.Error())
	}

	if err := c.Test(ctx, secretPending); err != nil {
		return errors.New("pending secret error: " + err.Error())
	}

	current := secretCurrent.(SecretUser)
	pending := secretPending.(SecretUser)

	if current[c.attributeKey] == pending[c.attributeKey] {
		return errors.New(`API key "` + c.attributeKey + `" shall be modified`)
	}

	if current[c.attributeSecret] == pending[c.attributeSecret] {
		return errors.New(`API secret "` + c.attributeSecret + `" shall be modified`)
	}

	current[c.attributeKey] = ""
	current[c.attributeSecret] = ""
	pending[c.attributeKey] = ""
	pending[c.attributeSecret] = ""

	if !reflect.DeepEqual(current, pending) {
		return errors.New("additional attributes of the current and pending secrets shall match")
	}

	return nil
}

func (c dbClient) Test(ctx context.Context, secret any) error {
	ctx = c.wrapContext(ctx)
	s, ok := secret.(SecretUser)
	if !ok {
		return errors.New("wrong secret type")
	}
	if _, ok := s[c.attributeKey]; !ok {
		return errors.New(`wrong secret type: "` + c.attributeKey + `" field not found`)
	}
	if _, ok := s[c.attributeSecret]; !ok {
		return errors.New(`wrong secret type: "` + c.attributeSecret + `" field not found`)
	}
	return nil
}

func (c dbClient) Create(ctx context.Context, secret any) error {
	ctx = c.wrapContext(ctx)

	s, ok := secret.(SecretUser)
	if !ok {
		return errors.New("wrong secret type")
	}

	id, ok := s[c.attributeKey]
	if !ok {
		return errors.New(`wrong secret type: "` + c.attributeKey + `" field not found`)
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

	s[c.attributeKey] = createdKey.GetId()

	sp, _ := createdKey.GetSpecOk()
	s[c.attributeSecret] = sp.GetSecret()

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
