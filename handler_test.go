package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"unsafe"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func Test_extractSecretObject(t *testing.T) {
	type args struct {
		v      *secretsmanager.GetSecretValueOutput
		secret any
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				v: &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{"password":"` + placeholderPassword + `"}`),
				},
				secret: &Secret{},
			},
			wantErr: false,
		},
		{
			name: "unhappy path",
			args: args{
				v: &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{`),
				},
				secret: &Secret{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if err := extractSecretObject(tt.args.v, tt.args.secret); (err != nil) != tt.wantErr {
					t.Errorf("extractSecretObject() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && tt.args.secret.(*Secret).Password != placeholderPassword {
					t.Errorf("extractSecretObject() failed to deserialize password")
				}
			},
		)
	}
}

type mockSecretsmanagerClient struct {
	secretAWSCurrent string

	secretByID map[string]map[string]string
}

func (m mockSecretsmanagerClient) getSecret(stage, version string) Secret {
	stages, ok := m.secretByID[version]
	if !ok {
		panic("no version " + version + " found")
	}

	s, ok := stages[stage]
	if !ok {
		panic("no stage " + stage + " for the version " + version + " found")
	}

	var secret Secret
	if err := json.Unmarshal(*(*[]byte)(unsafe.Pointer(&s)), &secret); err != nil {
		panic(err)
	}

	return secret
}

func (m mockSecretsmanagerClient) GetSecretValue(
	ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	o := &secretsmanager.GetSecretValueOutput{
		ARN: input.SecretId,
	}

	if input.VersionStage == nil && input.VersionId == nil {
		o.VersionStages = []string{"AWSCURRENT"}
		o.SecretString = &m.secretAWSCurrent
		return o, nil
	}

	stages, ok := m.secretByID[*input.VersionId]
	if !ok {
		return nil, errors.New("no version " + *input.VersionId + " found")
	}

	s, ok := stages[*input.VersionStage]
	if !ok {
		return nil, errors.New(
			"no stage " + *input.VersionStage + " for the version " + *input.VersionId + " found",
		)
	}

	stagesK := make([]string, len(stages))
	var i uint8
	for k := range stages {
		stagesK[i] = k
		i++
	}

	o.VersionStages = stagesK
	o.SecretString = &s

	return o, nil
}

func (m mockSecretsmanagerClient) PutSecretValue(
	ctx context.Context, input *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options),
) (*secretsmanager.PutSecretValueOutput, error) {
	return nil, nil
}

func (m mockSecretsmanagerClient) DescribeSecret(
	ctx context.Context, input *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options),
) (*secretsmanager.DescribeSecretOutput, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockSecretsmanagerClient) UpdateSecretVersionStage(
	ctx context.Context, input *secretsmanager.UpdateSecretVersionStageInput,
	optFns ...func(*secretsmanager.Options),
) (*secretsmanager.UpdateSecretVersionStageOutput, error) {
	//TODO implement me
	panic("implement me")
}

//	return &secretsmanager.GetSecretValueOutput{
//		ARN: input.SecretId,
//		SecretString: aws.String(
//			`{
//
// "dbname": "foo",
// "user": "bar",
// "host": "dev",
// "project_id": "baz",
// "branch_id": "br-foo",
// "password": "` + placeholderPassword + `"}`,
//
//		),
//		VersionId:      nil,
//		VersionStages:  nil,
//		ResultMetadata: middleware.Metadata{},
//	}, nil
func Test_createSecret(t *testing.T) {
	type args struct {
		ctx   context.Context
		event SecretsmanagerTriggerPayload
		cfg   Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "createSecret",
				},
				cfg: Config{
					SecretsmanagerClient: mockSecretsmanagerClient{},
					DBClient:             clientDB{c: newMockSDKClient()},
					SecretObj:            &Secret{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if err := createSecret(tt.args.ctx, tt.args.event, tt.args.cfg); (err != nil) != tt.wantErr {
					t.Errorf("createSecret() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}
