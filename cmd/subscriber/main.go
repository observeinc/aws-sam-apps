package main

import (
	"context"
	"fmt"

	awslambda "github.com/aws/aws-lambda-go/lambda"

	"github.com/observeinc/aws-sam-apps/pkg/lambda"
	"github.com/observeinc/aws-sam-apps/pkg/lambda/subscriber"
)

var (
	sub *subscriber.Lambda
)

func init() {
	ctx := context.Background()

	var config subscriber.Config
	err := lambda.ProcessEnv(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to initialize config: %w", err))
	}

	sub, err = subscriber.New(ctx, &config)
	if err != nil {
		panic(fmt.Errorf("failed to configure entrypoint: %w", err))
	}
}

func main() {
	awslambda.StartWithOptions(sub.Entrypoint, awslambda.WithEnableSIGTERM(sub.Shutdown))
}
