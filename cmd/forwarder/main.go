package main

import (
	"context"
	"fmt"

	awslambda "github.com/aws/aws-lambda-go/lambda"

	"github.com/observeinc/aws-sam-apps/pkg/lambda"
	"github.com/observeinc/aws-sam-apps/pkg/lambda/forwarder"
)

var (
	fwd *forwarder.Lambda
)

func init() {
	ctx := context.Background()

	var config forwarder.Config
	err := lambda.ProcessEnv(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to initialize config: %w", err))
	}

	fwd, err = forwarder.New(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to configure entrypoint: %w", err))
	}
}

func main() {
	awslambda.StartWithOptions(fwd.Entrypoint, awslambda.WithEnableSIGTERM(fwd.Shutdown))
}
