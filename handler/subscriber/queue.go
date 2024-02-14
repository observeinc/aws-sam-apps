package subscriber

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type queueWrapper struct {
	Client SQSClient
	URL    string
}

func (q *queueWrapper) Put(ctx context.Context, items ...*Request) error {
	for i, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item %d: %w", i, err)
		}

		_, err = q.Client.SendMessage(ctx, &sqs.SendMessageInput{
			MessageBody: aws.String(string(data)),
			QueueUrl:    aws.String(q.URL),
		})
		if err != nil {
			return fmt.Errorf("failed to send message %d: %w", i, err)
		}
	}
	return nil
}

func NewQueue(client SQSClient, queueURL string) (Queue, error) {
	q := &queueWrapper{
		Client: client,
		URL:    queueURL,
	}

	return q, nil
}
