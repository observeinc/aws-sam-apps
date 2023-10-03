package forwarder_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-testing/handlers/forwarder"
)

type MockS3Client struct {
	CopyObjectFunc func(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObjectFunc  func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func (c *MockS3Client) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	if c.CopyObjectFunc == nil {
		return nil, nil
	}
	return c.CopyObjectFunc(ctx, params, optFns...)
}

func (c *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if c.PutObjectFunc == nil {
		return nil, nil
	}
	return c.PutObjectFunc(ctx, params, optFns...)
}

func TestCopy(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		SourceURI      string
		DestinationURI string
		Expected       *s3.CopyObjectInput
	}{
		{
			SourceURI:      "s3://my-bucket/test.json",
			DestinationURI: "s3://another-bucket",
			Expected: &s3.CopyObjectInput{
				Bucket:     aws.String("another-bucket"),
				CopySource: aws.String("my-bucket/test.json"),
				Key:        aws.String("test.json"),
			},
		},
		{
			SourceURI:      "s3://my-bucket/hello/test.json",
			DestinationURI: "s3://another-bucket",
			Expected: &s3.CopyObjectInput{
				Bucket:     aws.String("another-bucket"),
				CopySource: aws.String("my-bucket/hello/test.json"),
				Key:        aws.String("hello/test.json"),
			},
		},
		{
			SourceURI:      "s3://my-bucket/hello/test.json",
			DestinationURI: "s3://another-bucket/prefix",
			Expected: &s3.CopyObjectInput{
				Bucket:     aws.String("another-bucket"),
				CopySource: aws.String("my-bucket/hello/test.json"),
				Key:        aws.String("prefix/hello/test.json"),
			},
		},
		{
			SourceURI:      "s3://my-bucket/hello/test.json",
			DestinationURI: "s3://another-bucket/prefix/",
			Expected: &s3.CopyObjectInput{
				Bucket:     aws.String("another-bucket"),
				CopySource: aws.String("my-bucket/hello/test.json"),
				Key:        aws.String("prefix/hello/test.json"),
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			s, err := url.ParseRequestURI(tc.SourceURI)
			if err != nil {
				t.Fatal(err)
			}
			d, err := url.ParseRequestURI(tc.DestinationURI)
			if err != nil {
				t.Fatal(err)
			}

			got := forwarder.GetCopyObjectInput(s, d)
			if diff := cmp.Diff(got, tc.Expected, cmp.AllowUnexported(s3.CopyObjectInput{})); diff != "" {
				t.Error("unexpected result", diff)
			}
		})
	}
}

var errSentinel = errors.New("sentinel error")

func TestHandler(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		RequestFile    string
		Config         forwarder.Config
		ExpectErr      error
		ExpectResponse events.SQSEventResponse
	}{
		{
			// Failing a copy should fail the individual item in the queue affected
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				DestinationURI: "s3://my-bucket",
				S3Client: &MockS3Client{
					CopyObjectFunc: func(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
						return nil, errSentinel
					},
				},
			},
			ExpectResponse: events.SQSEventResponse{
				BatchItemFailures: []events.SQSBatchItemFailure{
					{ItemIdentifier: "6aa4e980-26f6-46f4-bb73-fa6e82257191"},
				},
			},
		},
		{
			// Failing to put a record to the destination URI is a terminal condition. Error out.
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				DestinationURI: "s3://my-bucket",
				S3Client: &MockS3Client{
					PutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
						return nil, errSentinel
					},
				},
			},
			ExpectErr: errSentinel,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.RequestFile, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile("testdata/event.json")
			if err != nil {
				t.Fatal(err)
			}

			var request events.SQSEvent
			if err := json.Unmarshal(data, &request); err != nil {
				t.Fatal(err)
			}

			if tc.Config.Logger != nil {
				logger := testr.New(t)
				tc.Config.Logger = &logger
			}

			h, err := forwarder.New(&tc.Config)
			if err != nil {
				t.Fatal(err)
			}

			ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{})

			response, err := h.Handle(ctx, request)

			if diff := cmp.Diff(err, tc.ExpectErr, cmpopts.EquateErrors()); diff != "" {
				t.Error("unexpected error", diff)
			}

			if diff := cmp.Diff(response, tc.ExpectResponse); diff != "" {
				t.Error("unexpected response", diff)
			}
		})
	}
}
