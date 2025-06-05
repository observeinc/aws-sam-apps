package s3http_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func ptr[T any](v T) *T {
	return &v
}

func TestConfig(t *testing.T) {
	testcases := []struct {
		s3http.Config
		ExpectError error
	}{
		{
			ExpectError: s3http.ErrMissingS3Client,
		},
		{
			ExpectError: s3http.ErrInvalidDestination,
		},
		{
			Config: s3http.Config{
				DestinationURI:     "https://test",
				GetObjectAPIClient: &awstest.S3Client{},
			},
		},
		{
			Config: s3http.Config{
				DestinationURI:     "s3://test",
				GetObjectAPIClient: &awstest.S3Client{},
			},
			// S3 URI not supported
			ExpectError: s3http.ErrInvalidDestination,
		},
		{
			Config: s3http.Config{
				DestinationURI:     "https://test",
				GzipLevel:          ptr(200),
				GetObjectAPIClient: &awstest.S3Client{},
			},
			ExpectError: s3http.ErrUnsupportedGzipLevel,
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
