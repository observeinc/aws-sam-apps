package s3http_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/s3http"
	"github.com/observeinc/aws-sam-apps/handler/handlertest"
)

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
				GetObjectAPIClient: &handlertest.S3Client{},
			},
		},
		{
			Config: s3http.Config{
				DestinationURI:     "s3://test",
				GetObjectAPIClient: &handlertest.S3Client{},
			},
			// S3 URI not supported
			ExpectError: s3http.ErrInvalidDestination,
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
