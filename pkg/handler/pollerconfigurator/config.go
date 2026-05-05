package pollerconfigurator

import "github.com/observeinc/aws-sam-apps/pkg/logging"

type Config struct {
	ObserveAccountID  string `env:"OBSERVE_ACCOUNT_ID,required"`
	ObserveDomainName string `env:"OBSERVE_DOMAIN_NAME,required"`
	SecretName        string `env:"SECRET_NAME,required"`
	PollerConfigURI   string `env:"POLLER_CONFIG_URI,required"`
	ExternalRoleName  string `env:"EXTERNAL_ROLE_NAME,required"`
	WorkspaceID       string `env:"WORKSPACE_ID,required"`
	Region            string `env:"AWS_REGION,required"`
	AWSAccountID      string `env:"AWS_ACCOUNT_ID,required"`
	Logging           *logging.Config
}
