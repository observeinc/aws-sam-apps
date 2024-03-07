package forwarder_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/handler/forwarder"
	"github.com/observeinc/aws-sam-apps/handler/handlertest"
)

var lambdaContext = &lambdacontext.LambdaContext{
	InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:test",
	AwsRequestID:       "c8ee04d5-5925-541a-b113-5942a0fc5985",
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

	var copyFuncCallCount int

	testcases := []struct {
		RequestFile       string
		Config            forwarder.Config
		ExpectErr         error
		ExpectResponse    events.SQSEventResponse
		ExpectedCopyCalls int
	}{
		{
			// File size does not exceed MaxFileSize limit
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				MaxFileSize:       20,
				DestinationURI:    "s3://my-bucket",
				SourceBucketNames: []string{"observeinc*"},
				S3Client: &handlertest.S3Client{
					CopyObjectFunc: func(_ context.Context, _ *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
						return nil, nil
					},
				},
			},
			ExpectedCopyCalls: 1,
			ExpectResponse:    events.SQSEventResponse{
				// Expect no batch item failures as the file should be skipped, not failed
			},
		},
		{
			// File size exceeds MaxFileSize limit
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				MaxFileSize:       1,
				DestinationURI:    "s3://my-bucket",
				SourceBucketNames: []string{"observeinc*"},
				S3Client: &handlertest.S3Client{
					CopyObjectFunc: func(_ context.Context, _ *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
						return nil, nil
					},
				},
			},
			ExpectedCopyCalls: 0,
			ExpectResponse:    events.SQSEventResponse{
				// Expect no batch item failures as the file should be skipped, not failed
			},
		},
		{
			// Failing a copy should fail the individual item in the queue affected
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				DestinationURI:    "s3://my-bucket",
				SourceBucketNames: []string{"observeinc*"},
				S3Client: &handlertest.S3Client{
					CopyObjectFunc: func(_ context.Context, _ *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
						copyFuncCallCount++
						return nil, errSentinel
					},
				},
			},
			ExpectedCopyCalls: 1,
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
				DestinationURI:    "s3://my-bucket",
				SourceBucketNames: []string{"observeinc*"},
				S3Client: &handlertest.S3Client{
					PutObjectFunc: func(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
						return nil, errSentinel
					},
				},
			},
			ExpectedCopyCalls: 1,
			ExpectErr:         errSentinel,
		},
		{
			// Source bucket isn't in source bucket names
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				DestinationURI:    "s3://my-bucket",
				SourceBucketNames: []string{"doesntexist"},
				S3Client: &handlertest.S3Client{
					CopyObjectFunc: func(_ context.Context, _ *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
						return nil, nil
					},
				},
			},
			ExpectedCopyCalls: 0,
			ExpectResponse:    events.SQSEventResponse{
				// Expect no batch item failures as the file should be skipped, not failed
			},
		},
		{
			// Successful copy where source bucket matches a name in SourceBucketNames
			RequestFile: "testdata/event.json",
			Config: forwarder.Config{
				MaxFileSize:       50, // Adjust size limit to allow the file to be copied
				DestinationURI:    "s3://my-bucket",
				SourceBucketNames: []string{"doesntexist", "observeinc-filedrop-hoho-us-west-2-7xmjt"}, // List includes the exact bucket name
				S3Client: &handlertest.S3Client{
					CopyObjectFunc: func(_ context.Context, _ *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
						return nil, nil // Mock successful copy
					},
				},
			},
			ExpectedCopyCalls: 1, // Expect one successful call to CopyObjectFunc
			ExpectResponse:    events.SQSEventResponse{
				// Expect no batch item failures as the copy should be successful
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.RequestFile, func(t *testing.T) {
			t.Parallel()

			// Assert that S3Client is of the expected mock type
			mockS3Client, ok := tc.Config.S3Client.(*handlertest.S3Client)
			if !ok {
				t.Fatal("S3Client is not of type *handlertest.S3Client")
			}

			// Initialize the local counter for each test case
			localCopyFuncCallCount := 0

			// Save the original CopyObjectFunc
			originalCopyObjectFunc := mockS3Client.CopyObjectFunc

			// Wrap the CopyObjectFunc to increment the counter
			mockS3Client.CopyObjectFunc = func(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
				localCopyFuncCallCount++
				if originalCopyObjectFunc != nil {
					return originalCopyObjectFunc(ctx, params, optFns...)
				}
				return nil, nil // Or appropriate default behavior
			}

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

			ctx := lambdacontext.NewContext(context.Background(), lambdaContext)

			response, err := h.Handle(ctx, request)

			if diff := cmp.Diff(err, tc.ExpectErr, cmpopts.EquateErrors()); diff != "" {
				t.Error("unexpected error", diff)
			}

			if diff := cmp.Diff(response, tc.ExpectResponse); diff != "" {
				t.Error("unexpected response", diff)
			}

			// Assert the expected number of CopyObjectFunc calls
			if localCopyFuncCallCount != tc.ExpectedCopyCalls {
				t.Errorf("Expected CopyObjectFunc to be called %d times, but was called %d times", tc.ExpectedCopyCalls, localCopyFuncCallCount)
			}

			mockS3Client.CopyObjectFunc = originalCopyObjectFunc
		})
	}
}

func TestRecorder(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		DestinationURI string
		Now            func() time.Time
		Expect         *s3.PutObjectInput
	}{
		{
			DestinationURI: "s3://my-bucket/path/to",
			Now: func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2009-11-10T23:00:00Z")
				return t
			},
			Expect: &s3.PutObjectInput{
				Bucket:      aws.String("my-bucket"),
				Key:         aws.String("path/to/AWSLogs/123456789012/sqs/us-east-1/2009/11/10/23/c8ee04d5-5925-541a-b113-5942a0fc5985"),
				ContentType: aws.String("application/x-aws-sqs"),
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			var got *s3.PutObjectInput

			h, err := forwarder.New(&forwarder.Config{
				DestinationURI: tc.DestinationURI,
				S3Client: &handlertest.S3Client{
					PutObjectFunc: func(_ context.Context, i *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
						got = i
						return nil, nil
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			h.Now = tc.Now

			ctx := lambdacontext.NewContext(context.Background(), lambdaContext)
			if err := h.WriteSQS(ctx, nil); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(got, tc.Expect, cmpopts.IgnoreUnexported(s3.PutObjectInput{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
