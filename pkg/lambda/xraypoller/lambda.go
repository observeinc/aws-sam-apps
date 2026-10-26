package xraypoller

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/pkg/handler/xraypoller"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
	"github.com/observeinc/aws-sam-apps/pkg/version"
)

// Config is loaded from the Lambda environment.
type Config struct {
	CustomerID             string `env:"OBSERVE_CUSTOMER_ID,required"`
	Domain                 string `env:"OBSERVE_DOMAIN_NAME,required"`
	SecretID               string `env:"OBSERVE_INGEST_TOKEN_SECRET_ID,required"`
	Filter                 string `env:"FILTER_EXPRESSION"`
	CursorName             string `env:"CURSOR_PARAMETER_NAME,required"`
	SafetyLagSeconds       int    `env:"SAFETY_LAG_SECONDS,default=90"`
	InitialLookbackSeconds int    `env:"INITIAL_LOOKBACK_SECONDS,default=60"`
	MaxWindowSeconds       int    `env:"MAX_WINDOW_SECONDS,default=900"`
	Region                 string `env:"AWS_REGION"`
	Logging                *logging.Config
}

// Lambda is the EventBridge entrypoint.
type Lambda struct {
	Logger     logr.Logger
	Entrypoint func(context.Context, events.CloudWatchEvent) error
	Shutdown   func()
	poller     *xraypoller.Poller
}

// New wires AWS clients and the poller.
func New(ctx context.Context, cfg *Config) (*Lambda, error) {
	logger := logging.New(cfg.Logging)
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	if cfg.Region != "" {
		awsCfg.Region = cfg.Region
	}

	secrets := secretsmanager.NewFromConfig(awsCfg)
	store := &ssmCursor{client: ssm.NewFromConfig(awsCfg), name: cfg.CursorName}
	exporter := &xraypoller.HTTPExporter{
		URL:   xraypoller.Endpoint(cfg.CustomerID, cfg.Domain),
		Token: "",
	}
	poller := &xraypoller.Poller{
		XRay:            xray.NewFromConfig(awsCfg),
		Cursor:          store,
		Exporter:        &secretExporter{secrets: secrets, secretID: cfg.SecretID, inner: exporter},
		Region:          awsCfg.Region,
		ScopeVersion:    version.Version,
		Filter:          cfg.Filter,
		SafetyLag:       time.Duration(cfg.SafetyLagSeconds) * time.Second,
		InitialLookback: time.Duration(cfg.InitialLookbackSeconds) * time.Second,
		MaxWindow:       time.Duration(cfg.MaxWindowSeconds) * time.Second,
		Logger:          logger,
	}

	l := &Lambda{
		Logger: logger,
		poller: poller,
		Shutdown: func() {
			logger.V(4).Info("SIGTERM received, running shutdown")
		},
	}
	l.Entrypoint = l.handle
	return l, nil
}

func (l *Lambda) handle(ctx context.Context, event events.CloudWatchEvent) error {
	_, err := l.poller.Run(ctx, event.Time)
	return err
}

type secretExporter struct {
	secrets  *secretsmanager.Client
	secretID string
	inner    *xraypoller.HTTPExporter
}

func (s *secretExporter) Export(ctx context.Context, body []byte) error {
	out, err := s.secrets.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(s.secretID),
	})
	if err != nil {
		return fmt.Errorf("read ingest token: %w", err)
	}
	if out.SecretString == nil || *out.SecretString == "" {
		return fmt.Errorf("ingest token secret %s is empty", s.secretID)
	}
	s.inner.Token = *out.SecretString
	return s.inner.Export(ctx, body)
}

type ssmCursor struct {
	client *ssm.Client
	name   string
}

func (s *ssmCursor) Load(ctx context.Context) (time.Time, bool, error) {
	out, err := s.client.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(s.name)})
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read cursor: %w", err)
	}
	if out.Parameter == nil || out.Parameter.Value == nil {
		return time.Time{}, false, nil
	}
	return parseCursor(*out.Parameter.Value)
}

func (s *ssmCursor) Store(ctx context.Context, end time.Time) error {
	_, err := s.client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(s.name),
		Value:     aws.String(end.UTC().Format(time.RFC3339Nano)),
		Overwrite: aws.Bool(true),
		Type:      ssmtypes.ParameterTypeString,
	})
	if err != nil {
		return fmt.Errorf("write cursor: %w", err)
	}
	return nil
}

func parseCursor(value string) (time.Time, bool, error) {
	if value == "" || value == "0" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse cursor %q: %w", value, err)
	}
	return parsed, true, nil
}
