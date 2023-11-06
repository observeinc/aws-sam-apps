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

	"github.com/observeinc/aws-sam-testing/handler/forwarder"
	"github.com/observeinc/aws-sam-testing/handler/handlertest"
)

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
				S3Client: &handlertest.S3Client{
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
				S3Client: &handlertest.S3Client{
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
