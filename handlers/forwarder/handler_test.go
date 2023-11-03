package forwarder_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	HeadObjectFunc func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

func (c *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if c.HeadObjectFunc == nil {
		return nil, nil
	}
	return c.HeadObjectFunc(ctx, params, optFns...)
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
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				DestinationURI: "s3://my-bucket",
				SizeLimit:      1 * 1024 * 1024 * 1024, // 1 GB size limit for testing
				S3Client: &MockS3Client{
					HeadObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
						return &s3.HeadObjectOutput{
							ContentLength: 2 * 1024 * 1024 * 1024, // Set content length to 2 GB to trigger the error
						}, nil
					},
				},
			},
			ExpectErr: forwarder.ErrFileSizeLimitExceeded, // Expect the custom file size limit exceeded error
			ExpectResponse: events.SQSEventResponse{
				BatchItemFailures: []events.SQSBatchItemFailure{
					{ItemIdentifier: "6aa4e980-26f6-46f4-bb73-fa6e82257191"}, // Use the actual message ID you expect to fail
				},
			},
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

func TestRecorder(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Context        *lambdacontext.LambdaContext
		Prefix         string
		DestinationURI string
		Expect         *s3.PutObjectInput
	}{
		{
			Context: &lambdacontext.LambdaContext{
				InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:test",
				AwsRequestID:       "c8ee04d5-5925-541a-b113-5942a0fc5985",
			},
			Prefix:         "test/",
			DestinationURI: "s3://my-bucket/path/to",
			Expect: &s3.PutObjectInput{
				Bucket:      aws.String("my-bucket"),
				Key:         aws.String("path/to/test/arn:aws:lambda:us-east-1:123456789012:function:test/c8ee04d5-5925-541a-b113-5942a0fc5985"),
				ContentType: aws.String("application/x-ndjson"),
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			u, err := url.ParseRequestURI(tc.DestinationURI)
			if err != nil {
				t.Fatal(err)
			}

			var body io.Reader

			tc.Expect.Body = body
			got := forwarder.GetLogInput(tc.Context, tc.Prefix, u, body)

			if diff := cmp.Diff(got, tc.Expect, cmpopts.IgnoreUnexported(s3.PutObjectInput{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
