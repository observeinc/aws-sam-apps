package pollerconfigurator

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-logr/logr"
	"github.com/observeinc/aws-sam-apps/pkg/handler/pollerconfigurator"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
)

type Lambda struct {
	Logger     logr.Logger
	Entrypoint lambda.Handler
	Shutdown   func()
}

func New(ctx context.Context, cfg *pollerconfigurator.Config) (*Lambda, error) {
	logger := logging.New(cfg.Logging)
	logger.V(4).Info("initialized", "config", cfg)

	l := &Lambda{
		Logger: logger,
		Shutdown: func() {
			logger.V(4).Info("SIGTERM received, running shutdown")
		},
	}

	handler, err := pollerconfigurator.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	l.Entrypoint = handler
	return l, nil
}
