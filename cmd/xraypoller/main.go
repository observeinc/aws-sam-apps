package main

import (
	"context"
	"fmt"

	awslambda "github.com/aws/aws-lambda-go/lambda"

	"github.com/observeinc/aws-sam-apps/pkg/lambda"
	"github.com/observeinc/aws-sam-apps/pkg/lambda/xraypoller"
)

var entry *xraypoller.Lambda

func init() {
	ctx := context.Background()

	var config xraypoller.Config
	err := lambda.ProcessEnv(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to initialize config: %w", err))
	}

	entry, err = xraypoller.New(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to configure entrypoint: %w", err))
	}
}

func main() {
	awslambda.StartWithOptions(entry.Entrypoint, awslambda.WithEnableSIGTERM(entry.Shutdown))
}
