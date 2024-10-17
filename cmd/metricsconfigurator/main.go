package main

import (
	"context"
	"fmt"

	awslambda "github.com/aws/aws-lambda-go/lambda"

	handler "github.com/observeinc/aws-sam-apps/pkg/handler/metricsconfigurator"
	"github.com/observeinc/aws-sam-apps/pkg/lambda"
	"github.com/observeinc/aws-sam-apps/pkg/lambda/metricsconfigurator"
)

var (
	rec *metricsconfigurator.Lambda
)

func init() {
	ctx := context.Background()

	var config handler.Config
	err := lambda.ProcessEnv(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to initialize config: %w", err))
	}

	rec, err = metricsconfigurator.New(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to configure entrypoint: %w", err))
	}
}

func main() {
	awslambda.StartWithOptions(rec.Entrypoint, awslambda.WithEnableSIGTERM(rec.Shutdown))
}
