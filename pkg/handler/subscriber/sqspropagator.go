package subscriber

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"
)

type SQSMessageAttributesCarrier struct {
	typesMessageAttributes  map[string]types.MessageAttributeValue
	eventsMessageAttributes map[string]events.SQSMessageAttribute
	propagator              propagation.TextMapPropagator
}

func (c *SQSMessageAttributesCarrier) attachTypesAttributes(messageAttributes map[string]types.MessageAttributeValue) {
	if messageAttributes == nil {
		panic("messageAttributes map is nil")
	}
	c.typesMessageAttributes = messageAttributes
}

func (c *SQSMessageAttributesCarrier) attachEventsAttributes(messageAttributes map[string]events.SQSMessageAttribute) {
	if messageAttributes == nil {
		panic("messageAttributes map is nil")
	}
	c.eventsMessageAttributes = messageAttributes
}

func (c *SQSMessageAttributesCarrier) Extract(ctx context.Context, messageAttributes map[string]events.SQSMessageAttribute) context.Context {
	if messageAttributes == nil {
		return ctx
	}
	c.attachEventsAttributes(messageAttributes)
	return c.propagator.Extract(ctx, c)
}

func (c *SQSMessageAttributesCarrier) Inject(ctx context.Context, messageAttributes map[string]types.MessageAttributeValue) error {
	c.attachTypesAttributes(messageAttributes)
	c.propagator.Inject(ctx, c)
	return nil
}

// Get returns the value for the key.
func (c *SQSMessageAttributesCarrier) Get(key string) string {
	attr, found := c.eventsMessageAttributes[key]
	if !found {
		return ""
	}
	if attr.StringValue == nil {
		return ""
	}
	return *attr.StringValue
}

const stringType = "String"

// Set stores a key-value pair.
func (c *SQSMessageAttributesCarrier) Set(key, value string) {
	c.typesMessageAttributes[key] = types.MessageAttributeValue{
		DataType:    aws.String(stringType),
		StringValue: aws.String(value),
	}
}

// Keys lists the keys in the carrier.
func (c *SQSMessageAttributesCarrier) Keys() []string {
	keys := make([]string, 0, len(c.eventsMessageAttributes))
	for k := range c.eventsMessageAttributes {
		keys = append(keys, k)
	}
	return keys
}

func NewSQSCarrier() *SQSMessageAttributesCarrier {
	c := &SQSMessageAttributesCarrier{propagator: b3.New()}
	return c
}
