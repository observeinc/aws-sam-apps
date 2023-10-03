package forwarder_test

import (
	"fmt"
	"testing"

	"github.com/observeinc/aws-sam-testing/handlers/forwarder"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
				S3Client:       &MockS3Client{},
			},
		},
		{
			Config: forwarder.Config{
				DestinationURI: "https://example.com",
				S3Client:       &MockS3Client{},
			},
			ExpectError: forwarder.ErrInvalidDestination,
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
