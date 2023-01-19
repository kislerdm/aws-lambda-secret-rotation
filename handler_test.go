package lambda

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

func Test_extractSecretObject(t *testing.T) {
	type args struct {
		v      *secretsmanager.GetSecretValueOutput
		secret any
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantSecret any
	}{
		{
			name: "happy path",
			args: args{
				v: &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{"foo": "bar"}`),
				},
				secret: &map[string]string{},
			},
			wantErr:    false,
			wantSecret: &map[string]string{"foo": "bar"},
		},
		{
			name: "unhappy path",
			args: args{
				v: &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{`),
				},
				secret: nil,
			},
			wantErr:    true,
			wantSecret: nil,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if err := ExtractSecretObject(tt.args.v, tt.args.secret); (err != nil) != tt.wantErr {
					t.Errorf("ExtractSecretObject() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr {
					if !reflect.DeepEqual(tt.wantSecret, tt.args.secret) {
						t.Errorf("ExtractSecretObject() result does not match expectation")
					}
				}
			},
		)
	}
}

type mockSecretsmanagerClient struct {
	secretAWSCurrent string

	secretByID map[string]map[string]string

	rotationEnabled *bool
}

func getSecret(m *mockSecretsmanagerClient, stage, version string) SecretUser {
	stages, ok := m.secretByID[version]
	if !ok {
		panic("no version " + version + " found")
	}

	s, ok := stages[stage]
	if !ok {
		panic("no stage " + stage + " for the version " + version + " found")
	}

	var secret SecretUser
	if err := json.Unmarshal([]byte(s), &secret); err != nil {
		panic(err)
	}

	return secret
}

func (m *mockSecretsmanagerClient) GetSecretValue(
	ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	if m.secretAWSCurrent == "" {
		return nil, &types.ResourceNotFoundException{
			Message: aws.String("no secret found"),
		}
	}

	o := &secretsmanager.GetSecretValueOutput{
		ARN:           input.SecretId,
		VersionStages: []string{"AWSCURRENT"},
		SecretString:  &m.secretAWSCurrent,
	}

	if input.VersionId == nil || *input.VersionId == "" {
		if m.secretAWSCurrent == "" {
			return nil, &types.ResourceNotFoundException{
				Message: aws.String("no AWSCURRENT version found"),
			}
		}
		return o, nil
	}

	stages, ok := m.secretByID[*input.VersionId]
	if !ok {
		return nil, &types.ResourceNotFoundException{
			Message: aws.String("no version " + *input.VersionId + " found"),
		}
	}

	stage := *input.VersionStage
	if stage == "" {
		stage = "AWSCURRENT"
	}

	s, ok := stages[stage]
	if !ok {
		return nil, &types.ResourceNotFoundException{
			Message: aws.String("no stage " + stage + " for the version " + *input.VersionId + " found"),
		}
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

func (m *mockSecretsmanagerClient) PutSecretValue(
	ctx context.Context, input *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options),
) (*secretsmanager.PutSecretValueOutput, error) {
	versionID := *input.ClientRequestToken
	stage := input.VersionStages[0]

	if m.secretByID == nil {
		m.secretByID = map[string]map[string]string{}
	}

	if _, ok := m.secretByID[versionID]; !ok {
		m.secretByID[versionID] = map[string]string{}
	}

	m.secretByID[versionID][stage] = *input.SecretString

	return nil, nil
}

func (m *mockSecretsmanagerClient) DescribeSecret(
	ctx context.Context, input *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options),
) (*secretsmanager.DescribeSecretOutput, error) {
	if m.secretAWSCurrent == "" {
		return nil, errors.New("no secret found")
	}

	if m.secretByID == nil {
		return &secretsmanager.DescribeSecretOutput{
			ARN: input.SecretId,
		}, nil
	}

	versionIdsToStages := make(map[string][]string, len(m.secretByID))
	for k, v := range m.secretByID {
		versionIdsToStages[k] = make([]string, len(v))
		var i uint8
		for s := range v {
			versionIdsToStages[k][i] = s
			i++
		}
	}

	return &secretsmanager.DescribeSecretOutput{
		ARN:                input.SecretId,
		VersionIdsToStages: versionIdsToStages,
		RotationEnabled:    m.rotationEnabled,
	}, nil
}

func (m *mockSecretsmanagerClient) UpdateSecretVersionStage(
	ctx context.Context, input *secretsmanager.UpdateSecretVersionStageInput,
	optFns ...func(*secretsmanager.Options),
) (*secretsmanager.UpdateSecretVersionStageOutput, error) {
	m.secretAWSCurrent = m.secretByID[*input.RemoveFromVersionId]["AWSPENDING"]
	m.secretByID[*input.RemoveFromVersionId]["AWSCURRENT"] = m.secretAWSCurrent
	delete(m.secretByID[*input.RemoveFromVersionId], "AWSPENDING")
	return nil, nil
}

var (
	placeholderSecretUserStr = `{"user":"bar","password":"` + placeholderPassword +
		`","host":"dev","project_id":"baz","branch_id":"br-foo","dbname":"foo"}`

	placeholderSecretUserNewStr = `{"user":"bar","password":"` + placeholderPassword +
		`new","host":"dev","project_id":"baz","branch_id":"br-foo","dbname":"foo"}`

	placeholderSecretUser = SecretUser{
		User:         "bar",
		Password:     placeholderPassword,
		Host:         "dev",
		ProjectID:    "baz",
		BranchID:     "br-foo",
		DatabaseName: "foo",
	}
	placeholderSecretUserNew = SecretUser{
		User:         "bar",
		Password:     placeholderPassword + "new",
		Host:         "dev",
		ProjectID:    "baz",
		BranchID:     "br-foo",
		DatabaseName: "foo",
	}
)

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
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: false,
		},
		{
			name: "happy path: new secret already in the pending stage",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "createSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
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

				if !tt.wantErr {
					secretInitial := placeholderSecretUser
					passwordInitial := secretInitial.Password
					secretInitial.Password = ""

					secretNew := getSecret(
						tt.args.cfg.SecretsmanagerClient.(*mockSecretsmanagerClient),
						"AWSPENDING",
						tt.args.event.Token,
					)
					passwordNew := secretNew.Password
					secretNew.Password = ""

					if passwordNew == passwordInitial || !reflect.DeepEqual(secretInitial, secretNew) {
						t.Errorf("generated secret does not match expectation")
					}
				}
			},
		)
	}
}

func Test_serialiseSecret(t *testing.T) {
	type args struct {
		secret any
	}
	tests := []struct {
		name    string
		args    args
		want    *string
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				secret: placeholderSecretUser,
			},
			want:    &placeholderSecretUserStr,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := serialiseSecret(tt.args.secret)
				if (err != nil) != tt.wantErr {
					t.Errorf("serialiseSecret() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("serialiseSecret() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_finishSecret(t *testing.T) {
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
					Step:      "finishSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: false,
		},
		{
			name: "happy path: already set",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "finishSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserNewStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserNewStr,
							},
						},
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if err := finishSecret(tt.args.ctx, tt.args.event, tt.args.cfg); (err != nil) != tt.wantErr {
					t.Errorf("finishSecret() error = %v, wantErr %v", err, tt.wantErr)
				}

				if !tt.wantErr {
					if !reflect.DeepEqual(
						getSecret(
							tt.args.cfg.SecretsmanagerClient.(*mockSecretsmanagerClient),
							"AWSCURRENT",
							"foo",
						),
						placeholderSecretUserNew,
					) {
						t.Errorf("finishSecret() result does not match expectation")
					}

					if tt.args.cfg.SecretsmanagerClient.(*mockSecretsmanagerClient).secretAWSCurrent !=
						placeholderSecretUserNewStr {
						t.Errorf("finishSecret() result does not match expectation")
					}
				}
			},
		)
	}
}

func Test_setSecret(t *testing.T) {
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
					Step:      "setSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: false,
		},
		{
			name: "unhappy path: no AWSCURRENT version",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "setSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{},
					DBClient:             dbClient{c: newMockSDKClient()},
					SecretObj:            &SecretUser{},
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: no AWSPENDING version",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "setSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
							},
						},
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if err := setSecret(tt.args.ctx, tt.args.event, tt.args.cfg); (err != nil) != tt.wantErr {
					t.Errorf("setSecret() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}

func Test_testSecret(t *testing.T) {
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
					Step:      "testSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: false,
		},
		{
			name: "unhappy path: no AWSPENDING found",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "testSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: faulty new secret value",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "testSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSPENDING": `{`,
							},
						},
					},
					DBClient:  dbClient{c: newMockSDKClient()},
					SecretObj: &SecretUser{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if err := testSecret(tt.args.ctx, tt.args.event, tt.args.cfg); (err != nil) != tt.wantErr {
					t.Errorf("testSecret() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}

func Test_validateEvent(t *testing.T) {
	type args struct {
		ctx    context.Context
		event  SecretsmanagerTriggerPayload
		client SecretsmanagerClient
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errType error
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
				client: &mockSecretsmanagerClient{
					secretAWSCurrent: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					rotationEnabled:  aws.Bool(true),
					secretByID: map[string]map[string]string{
						"foo": {
							"AWSPENDING": placeholderSecretUserStr,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "unhappy path: no secret exists",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "createSecret",
				},
				client: &mockSecretsmanagerClient{},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: rotation is not enabled",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "createSecret",
				},
				client: &mockSecretsmanagerClient{
					secretAWSCurrent: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					secretByID: map[string]map[string]string{
						"foo": {
							"AWSPENDING": placeholderSecretUserStr,
						},
					},
					rotationEnabled: aws.Bool(false),
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: no stages for the version",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "createSecret",
				},
				client: &mockSecretsmanagerClient{
					secretAWSCurrent: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					secretByID: map[string]map[string]string{
						"foo": {
							"AWSPENDING": placeholderSecretUserStr,
						},
					},
					rotationEnabled: aws.Bool(true),
				},
			},
			wantErr: true,
		},
		{
			name: "happy path: AWSCURRENT is present",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "createSecret",
				},
				client: &mockSecretsmanagerClient{
					secretAWSCurrent: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					secretByID: map[string]map[string]string{
						"foo": {
							"AWSCURRENT": placeholderSecretUserStr,
						},
					},
					rotationEnabled: aws.Bool(true),
				},
			},
			wantErr: true,
			errType: os.ErrExist,
		},
		{
			name: "unhappy path: AWSPENDING is not present",
			args: args{
				ctx: context.TODO(),
				event: SecretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "createSecret",
				},
				client: &mockSecretsmanagerClient{
					secretAWSCurrent: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					secretByID: map[string]map[string]string{
						"foo": {
							"AWSPREVIOUS": placeholderSecretUserStr,
						},
					},
					rotationEnabled: aws.Bool(true),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				err := validateEvent(tt.args.ctx, tt.args.event, tt.args.client)
				if (err != nil) != tt.wantErr {
					t.Errorf("validateEvent() error = %v, wantErr %v", err, tt.wantErr)

					if tt.errType != nil {
						if !errors.Is(err, tt.errType) {
							t.Errorf("validateEvent() returned error type does not match expectation")
						}
					}
				}
			},
		)
	}
}
