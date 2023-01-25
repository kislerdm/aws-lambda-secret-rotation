package neon

import (
	"context"

	sdk "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	lambda "github.com/kislerdm/aws-lambda-secret-rotation"
)

// NewServiceClient initiates the `ServiceClient` to rotate credentials for Confluent Kafka user.
func NewServiceClient(client *sdk.APIClient) lambda.ServiceClient {
	return &dbClient{c: client}
}

type dbClient struct {
	c *sdk.APIClient
}

func (c dbClient) Set(ctx context.Context, secretCurrent, secretPending, secretPrevious any) error {
	panic("todo")
}

func (c dbClient) Test(ctx context.Context, secret any) error {
	panic("todo")
}

func (c dbClient) Create(ctx context.Context, secret any) error {
	panic("todo")
}
