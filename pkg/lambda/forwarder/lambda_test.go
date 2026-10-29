package forwarder

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type secretsManagerClientFunc func(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)

func (f secretsManagerClientFunc) GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return f(ctx, input, opts...)
}

func TestGetAuthorizationHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		secret    string
		clientErr error
		want      string
		wantErr   bool
	}{
		{name: "disabled"},
		{name: "valid", secret: `{"username":"user","password":"pass"}`, want: "Basic dXNlcjpwYXNz"},
		{name: "malformed", secret: `{`, wantErr: true},
		{name: "missing password", secret: `{"username":"user"}`, wantErr: true},
		{name: "client failure", clientErr: errors.New("denied"), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			secretARN := "arn:aws:secretsmanager:us-west-2:123456789012:secret:example"
			if test.name == "disabled" {
				secretARN = ""
			}
			client := secretsManagerClientFunc(func(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				if aws.ToString(input.SecretId) != secretARN {
					t.Errorf("unexpected secret ARN %q", aws.ToString(input.SecretId))
				}
				return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(test.secret)}, test.clientErr
			})
			got, err := getAuthorizationHeader(context.Background(), client, secretARN)
			if (err != nil) != test.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
