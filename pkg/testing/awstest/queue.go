package awstest

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient struct {
	SendMessageFunc func(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func (c *SQSClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return c.SendMessageFunc(ctx, params, optFns...)
}
