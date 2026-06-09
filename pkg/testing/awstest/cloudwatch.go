package awstest

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// MetricStreamRecord is a fake metric stream entry used by CloudWatchClient.
// Tags are stored on the record so tests can verify tag-based selection
// without a separate tag store.
type MetricStreamRecord struct {
	Name string
	Arn  string
	Tags []types.Tag
}

// CloudWatchClient is a fake cloudwatch.Client suitable for unit tests.
// It implements the subset of the SDK surface that pkg/handler/metricsconfigurator
// uses: PutMetricStream, ListMetricStreams (with pagination), ListTagsForResource,
// and DeleteMetricStream.
//
// Streams holds the in-memory state of metric streams. PutMetricStream
// appends or replaces by Name. DeleteMetricStream removes by Name.
//
// PageSize controls the number of entries ListMetricStreams returns per
// page (0 means "all in one page").
//
// Each method has an optional Func override for tests that need to inject
// errors or custom behavior.
type CloudWatchClient struct {
	Streams  []MetricStreamRecord
	PageSize int

	PutMetricStreamFunc     func(context.Context, *cloudwatch.PutMetricStreamInput, ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricStreamOutput, error)
	ListMetricStreamsFunc   func(context.Context, *cloudwatch.ListMetricStreamsInput, ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricStreamsOutput, error)
	ListTagsForResourceFunc func(context.Context, *cloudwatch.ListTagsForResourceInput, ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error)
	DeleteMetricStreamFunc  func(context.Context, *cloudwatch.DeleteMetricStreamInput, ...func(*cloudwatch.Options)) (*cloudwatch.DeleteMetricStreamOutput, error)
}

func (c *CloudWatchClient) PutMetricStream(ctx context.Context, input *cloudwatch.PutMetricStreamInput, opts ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricStreamOutput, error) {
	if c.PutMetricStreamFunc != nil {
		return c.PutMetricStreamFunc(ctx, input, opts...)
	}
	if input.Name == nil {
		return nil, errors.New("PutMetricStream: Name is required")
	}
	rec := MetricStreamRecord{
		Name: aws.ToString(input.Name),
		Arn:  "arn:aws:cloudwatch:us-west-2:000000000000:metric-stream/" + aws.ToString(input.Name),
		Tags: input.Tags,
	}
	for i, s := range c.Streams {
		if s.Name == rec.Name {
			c.Streams[i] = rec
			return &cloudwatch.PutMetricStreamOutput{Arn: aws.String(rec.Arn)}, nil
		}
	}
	c.Streams = append(c.Streams, rec)
	return &cloudwatch.PutMetricStreamOutput{Arn: aws.String(rec.Arn)}, nil
}

func (c *CloudWatchClient) ListMetricStreams(ctx context.Context, input *cloudwatch.ListMetricStreamsInput, opts ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricStreamsOutput, error) {
	if c.ListMetricStreamsFunc != nil {
		return c.ListMetricStreamsFunc(ctx, input, opts...)
	}

	start := 0
	if input.NextToken != nil {
		// Tokens we hand out are zero-based start indexes encoded as strings.
		for i, s := range c.Streams {
			if s.Name == aws.ToString(input.NextToken) {
				start = i
				break
			}
		}
	}

	end := len(c.Streams)
	if c.PageSize > 0 && start+c.PageSize < end {
		end = start + c.PageSize
	}

	entries := make([]types.MetricStreamEntry, 0, end-start)
	for _, s := range c.Streams[start:end] {
		name := s.Name
		arn := s.Arn
		entries = append(entries, types.MetricStreamEntry{Name: &name, Arn: &arn})
	}

	var next *string
	if end < len(c.Streams) {
		// Use the next stream's name as the page token.
		token := c.Streams[end].Name
		next = &token
	}
	return &cloudwatch.ListMetricStreamsOutput{Entries: entries, NextToken: next}, nil
}

func (c *CloudWatchClient) ListTagsForResource(ctx context.Context, input *cloudwatch.ListTagsForResourceInput, opts ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error) {
	if c.ListTagsForResourceFunc != nil {
		return c.ListTagsForResourceFunc(ctx, input, opts...)
	}
	for _, s := range c.Streams {
		if s.Arn == aws.ToString(input.ResourceARN) {
			tags := make([]types.Tag, len(s.Tags))
			copy(tags, s.Tags)
			return &cloudwatch.ListTagsForResourceOutput{Tags: tags}, nil
		}
	}
	return &cloudwatch.ListTagsForResourceOutput{}, nil
}

func (c *CloudWatchClient) DeleteMetricStream(ctx context.Context, input *cloudwatch.DeleteMetricStreamInput, opts ...func(*cloudwatch.Options)) (*cloudwatch.DeleteMetricStreamOutput, error) {
	if c.DeleteMetricStreamFunc != nil {
		return c.DeleteMetricStreamFunc(ctx, input, opts...)
	}
	target := aws.ToString(input.Name)
	for i, s := range c.Streams {
		if s.Name == target {
			c.Streams = append(c.Streams[:i], c.Streams[i+1:]...)
			return &cloudwatch.DeleteMetricStreamOutput{}, nil
		}
	}
	return &cloudwatch.DeleteMetricStreamOutput{}, nil
}
