package lambda

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/smithy-go"
	smithyHttp "github.com/aws/smithy-go/transport/http"
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

type mockObj struct {
	User         string `json:"user"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	ProjectID    string `json:"project_id"`
	BranchID     string `json:"branch_id"`
	DatabaseName string `json:"dbname"`
}

type mockSecretsmanagerClient struct {
	secretAWSCurrent  string
	secretAWSPrevious string

	secretByID map[string]map[string]string

	rotationEnabled *bool
}

func getSecret(m *mockSecretsmanagerClient, stage, version string) mockObj {
	stages, ok := m.secretByID[version]
	if !ok {
		panic("no version " + version + " found")
	}

	s, ok := stages[stage]
	if !ok {
		panic("no stage " + stage + " for the version " + version + " found")
	}

	var secret mockObj
	if err := json.Unmarshal([]byte(s), &secret); err != nil {
		panic(err)
	}

	return secret
}

func (m *mockSecretsmanagerClient) GetSecretValue(
	ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	if *input.VersionStage == "AWSPREVIOUS" {
		if m.secretAWSPrevious == "" {
			return nil, &smithy.OperationError{
				ServiceID:     "SecretsManager",
				OperationName: "GetSecretValue",
				Err: &smithyHttp.ResponseError{
					Response: &smithyHttp.Response{
						Response: &http.Response{
							StatusCode: http.StatusBadRequest,
						},
					},
					Err: errors.New("no secret found"),
				},
			}
		}
		return &secretsmanager.GetSecretValueOutput{
			ARN:          input.SecretId,
			SecretString: &m.secretAWSPrevious,
		}, nil
	}

	if m.secretAWSCurrent == "" {
		return nil, &smithy.OperationError{
			ServiceID:     "SecretsManager",
			OperationName: "GetSecretValue",
			Err: &smithyHttp.ResponseError{
				Response: &smithyHttp.Response{
					Response: &http.Response{
						StatusCode: http.StatusBadRequest,
					},
				},
				Err: errors.New("no secret found"),
			},
		}
	}

	o := &secretsmanager.GetSecretValueOutput{
		ARN:           input.SecretId,
		VersionStages: []string{"AWSCURRENT"},
		SecretString:  &m.secretAWSCurrent,
	}

	if input.VersionId == nil || *input.VersionId == "" {
		if m.secretAWSCurrent == "" {
			return nil, &smithy.OperationError{
				ServiceID:     "SecretsManager",
				OperationName: "GetSecretValue",
				Err: &smithyHttp.ResponseError{
					Response: &smithyHttp.Response{
						Response: &http.Response{
							StatusCode: http.StatusBadRequest,
						},
					},
					Err: errors.New("no secret found"),
				},
			}
		}

		return o, nil
	}

	stages, ok := m.secretByID[*input.VersionId]
	if !ok {
		return nil, &smithy.OperationError{
			ServiceID:     "SecretsManager",
			OperationName: "GetSecretValue",
			Err: &smithyHttp.ResponseError{
				Response: &smithyHttp.Response{
					Response: &http.Response{
						StatusCode: http.StatusBadRequest,
					},
				},
				Err: errors.New("no version " + *input.VersionId + " found"),
			},
		}
	}

	stage := *input.VersionStage
	if stage == "" {
		stage = "AWSCURRENT"
	}

	s, ok := stages[stage]
	if !ok {
		return nil, &smithy.OperationError{
			ServiceID:     "SecretsManager",
			OperationName: "GetSecretValue",
			Err: &smithyHttp.ResponseError{
				Response: &smithyHttp.Response{
					Response: &http.Response{
						StatusCode: http.StatusBadRequest,
					},
				},
				Err: errors.New("no stage " + stage + " for the version " + *input.VersionId + " found"),
			},
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
	m.secretAWSCurrent = m.secretByID[*input.MoveToVersionId]["AWSPENDING"]
	m.secretByID[*input.MoveToVersionId]["AWSCURRENT"] = m.secretAWSCurrent
	delete(m.secretByID[*input.MoveToVersionId], "AWSPENDING")
	delete(m.secretByID[*input.RemoveFromVersionId], "AWSCURRENT")
	return nil, nil
}

var (
	placeholderPassword      = "quxx"
	placeholderSecretUserStr = `{"user":"bar","password":"` + placeholderPassword +
		`","host":"dev","project_id":"baz","branch_id":"br-foo","dbname":"foo"}`

	placeholderSecretUserNewStr = `{"user":"bar","password":"` + placeholderPassword +
		`new","host":"dev","project_id":"baz","branch_id":"br-foo","dbname":"foo"}`

	placeholderSecretUser = mockObj{
		User:         "bar",
		Password:     placeholderPassword,
		Host:         "dev",
		ProjectID:    "baz",
		BranchID:     "br-foo",
		DatabaseName: "foo",
	}
	placeholderSecretUserNew = mockObj{
		User:         "bar",
		Password:     placeholderPassword + "new",
		Host:         "dev",
		ProjectID:    "baz",
		BranchID:     "br-foo",
		DatabaseName: "foo",
	}
)

type mockDBClient struct {
	current, pending, previous any
}

func (m *mockDBClient) Set(ctx context.Context, secretCurrent, secretPending, secretPrevious any) error {
	m.current = secretCurrent
	m.pending = secretPending
	m.previous = secretPrevious
	return nil
}

func (m *mockDBClient) Test(ctx context.Context, secret any) error {
	return nil
}

func (m *mockDBClient) Create(ctx context.Context, secret any) error {
	secret.(*mockObj).Password = placeholderSecretUserNewStr
	return nil
}

func Test_createSecret(t *testing.T) {
	type args struct {
		ctx   context.Context
		event secretsmanagerTriggerPayload
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
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "createSecret",
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
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
				},
			},
			wantErr: false,
		},
		{
			name: "happy path: new secret already in the pending stage",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "createSecret",
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
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
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
		event secretsmanagerTriggerPayload
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
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "finishSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
							},
							"bar": {
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
				},
			},
			wantErr: false,
		},
		{
			name: "happy path: already set",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "finishSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserNewStr,
						secretByID: map[string]map[string]string{
							"bar": {
								"AWSCURRENT": placeholderSecretUserNewStr,
							},
						},
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
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
							"bar",
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

type mapType map[string]string

func Test_setSecret(t *testing.T) {
	var mType mapType
	type args struct {
		ctx   context.Context
		event secretsmanagerTriggerPayload
		cfg   Config
	}
	tests := []struct {
		name                string
		args                args
		wantErr             bool
		wantExpectedCurrent any
		wantExpectedPending any
	}{
		{
			name: "happy path",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "setSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT":  placeholderSecretUserStr,
								"AWSPREVIOUS": placeholderSecretUserStr,
							},
							"bar": {
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
				},
			},
			wantErr:             false,
			wantExpectedCurrent: &placeholderSecretUser,
			wantExpectedPending: &placeholderSecretUserNew,
		},
		{
			name: "happy path: SecretObj-map",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "setSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: `{"foo": "bar"}`,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": `{"foo": "bar"}`,
							},
							"bar": {
								"AWSPENDING": `{"foo": "baz"}`,
							},
						},
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &mType,
					Debug:         true,
				},
			},
			wantErr:             false,
			wantExpectedCurrent: &mapType{"foo": "bar"},
			wantExpectedPending: &mapType{"foo": "baz"},
		},
		{
			name: "happy path: AWSPREVIOUS is present",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "bar",
					Step:      "setSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent:  placeholderSecretUserStr,
						secretAWSPrevious: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
							},
							"bar": {
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
				},
			},
			wantErr: false,
		},
		{
			name: "happy path: no AWSCURRENT version",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "setSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{},
					ServiceClient:        &mockDBClient{},
					SecretObj:            &mockObj{},
					Debug:                true,
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: no AWSPENDING version",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
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
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
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
				if !tt.wantErr {
					m := tt.args.cfg.ServiceClient.(*mockDBClient)
					if tt.wantExpectedCurrent != nil {
						if !reflect.DeepEqual(m.current, tt.wantExpectedCurrent) {
							t.Errorf("setSecret() current secret is not propagated right")
						}
					}
					if tt.wantExpectedPending != nil {
						if !reflect.DeepEqual(m.pending, tt.wantExpectedPending) {
							t.Errorf("setSecret() pending secret is not propagated right")
						}
					}
				}
			},
		)
	}
}

func Test_testSecret(t *testing.T) {
	type args struct {
		ctx   context.Context
		event secretsmanagerTriggerPayload
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
				event: secretsmanagerTriggerPayload{
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
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
				},
			},
			wantErr: false,
		},
		{
			name: "unhappy path: no AWSPENDING found",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "testSecret",
				},
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
				},
			},
			wantErr: true,
		},
		{
			name: "unhappy path: faulty new secret value",
			args: args{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
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
					ServiceClient: &mockDBClient{},
					SecretObj:     &mockObj{},
					Debug:         true,
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
		event  secretsmanagerTriggerPayload
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
				event: secretsmanagerTriggerPayload{
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
				event: secretsmanagerTriggerPayload{
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
				event: secretsmanagerTriggerPayload{
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
				event: secretsmanagerTriggerPayload{
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
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				err := validateInput(tt.args.ctx, tt.args.event, tt.args.client)
				if (err != nil) != tt.wantErr {
					t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)

					if tt.errType != nil {
						if !errors.Is(err, tt.errType) {
							t.Errorf("validateInput() returned error type does not match expectation")
						}
					}
				}
			},
		)
	}
}

func TestStrToBool(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "positive",
			args: args{
				s: "yes",
			},
			want: true,
		},
		{
			name: "positive",
			args: args{
				s: "y",
			},
			want: true,
		},
		{
			name: "positive",
			args: args{
				s: "true",
			},
			want: true,
		},
		{
			name: "positive",
			args: args{
				s: "1",
			},
			want: true,
		},
		{
			name: "negative",
			args: args{
				s: "no",
			},
			want: false,
		},
		{
			name: "negative",
			args: args{
				s: "n",
			},
			want: false,
		},
		{
			name: "negative",
			args: args{
				s: "false",
			},
			want: false,
		},
		{
			name: "negative",
			args: args{
				s: "0",
			},
			want: false,
		},
	}

	t.Parallel()
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				for _, fn := range []func(string) string{strings.ToLower, strings.ToUpper} {
					s := fn(tt.args.s)
					if got := StrToBool(s); got != tt.want {
						t.Errorf("StrToBool() = %v, want %v", got, tt.want)
					}
				}
			},
		)
	}
}

func TestNewHandler(t *testing.T) {
	type args struct {
		cfg Config
	}
	type argsHandler struct {
		ctx   context.Context
		event secretsmanagerTriggerPayload
	}
	tests := []struct {
		name        string
		args        args
		argsHandler argsHandler
		wantErrInit bool
		wantErr     bool
	}{
		{
			name: "unhappy path: SecretObj set to nil",
			args: args{
				cfg: Config{},
			},
			argsHandler: argsHandler{},
			wantErrInit: true,
			wantErr:     false,
		},
		{
			name: "unhappy path: unknown step",
			args: args{
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
							},
						},
						rotationEnabled: aws.Bool(true),
					},
					SecretObj: &map[string]string{},
					Debug:     true,
				},
			},
			argsHandler: argsHandler{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "foobar",
				},
			},
			wantErrInit: false,
			wantErr:     true,
		},
		{
			name: "unhappy path: does not pass input validation",
			args: args{
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
							},
						},
					},
					SecretObj: &map[string]string{},
					Debug:     true,
				},
			},
			argsHandler: argsHandler{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "foobar",
				},
			},
			wantErrInit: false,
			wantErr:     true,
		},
		{
			name: "happy path: createSecret step",
			args: args{
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
						rotationEnabled: aws.Bool(true),
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &map[string]string{},
					Debug:         true,
				},
			},
			argsHandler: argsHandler{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "createSecret",
				},
			},
			wantErrInit: false,
			wantErr:     false,
		},
		{
			name: "happy path: setSecret step",
			args: args{
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
						rotationEnabled: aws.Bool(true),
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &map[string]string{},
					Debug:         true,
				},
			},
			argsHandler: argsHandler{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "setSecret",
				},
			},
			wantErrInit: false,
			wantErr:     false,
		},
		{
			name: "happy path: testSecret step",
			args: args{
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
						rotationEnabled: aws.Bool(true),
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &map[string]string{},
					Debug:         true,
				},
			},
			argsHandler: argsHandler{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "testSecret",
				},
			},
			wantErrInit: false,
			wantErr:     false,
		},
		{
			name: "happy path: finishSecret step",
			args: args{
				cfg: Config{
					SecretsmanagerClient: &mockSecretsmanagerClient{
						secretAWSCurrent: placeholderSecretUserStr,
						secretByID: map[string]map[string]string{
							"foo": {
								"AWSCURRENT": placeholderSecretUserStr,
								"AWSPENDING": placeholderSecretUserNewStr,
							},
						},
						rotationEnabled: aws.Bool(true),
					},
					ServiceClient: &mockDBClient{},
					SecretObj:     &map[string]string{},
					Debug:         true,
				},
			},
			argsHandler: argsHandler{
				ctx: context.TODO(),
				event: secretsmanagerTriggerPayload{
					SecretARN: "arn:aws:secretsmanager:us-east-1:000000000000:secret:foo/bar-5BKPC8",
					Token:     "foo",
					Step:      "finishSecret",
				},
			},
			wantErrInit: false,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				handler, err := NewHandler(tt.args.cfg)
				if (err != nil) != tt.wantErrInit {
					t.Errorf("NewHandler() error = %v, wantErrInit %v", err, tt.wantErrInit)
					return
				}
				if !tt.wantErrInit {
					if err := handler(tt.argsHandler.ctx, tt.argsHandler.event); (err != nil) != tt.wantErr {
						t.Errorf("handler(ctx, event) error = %v, wantErr %v", err, tt.wantErr)
						return
					}
				}
			},
		)
	}
}
