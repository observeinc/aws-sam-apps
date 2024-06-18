package forwarder_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func TestConfig(t *testing.T) {
	testcases := []struct {
		forwarder.Config
		ExpectError error
	}{
		{
			ExpectError: forwarder.ErrMissingS3Client,
		},
		{
			ExpectError: forwarder.ErrInvalidDestination,
		},
		{
			Config: forwarder.Config{
				DestinationURI: "s3://test",
				S3Client:       &awstest.S3Client{},
			},
		},
		{
			Config: forwarder.Config{
				DestinationURI: "ftp://example.com",
				S3Client:       &awstest.S3Client{},
			},
			ExpectError: forwarder.ErrInvalidDestination,
		},
		{
			Config: forwarder.Config{
				DestinationURI:    "https://example.com",
				S3Client:          &awstest.S3Client{},
				SourceBucketNames: []string{"bucket*"},
				SourceObjectKeys:  []string{"*/te?t/*"},
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := tc.Validate()
			if diff := cmp.Diff(err, tc.ExpectError, cmpopts.EquateErrors()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
