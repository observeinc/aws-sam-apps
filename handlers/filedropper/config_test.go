package filedropper_test

import (
	"fmt"
	"testing"

	"github.com/observeinc/aws-sam-testing/handlers/filedropper"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestConfig(t *testing.T) {
	testcases := []struct {
		filedropper.Config
		ExpectError error
	}{
		{
			ExpectError: filedropper.ErrMissingS3Client,
		},
		{
			ExpectError: filedropper.ErrInvalidDestination,
		},
		{
			Config: filedropper.Config{
				DestinationURI: "s3://test",
				S3Client:       &MockS3Client{},
			},
		},
		{
			Config: filedropper.Config{
				DestinationURI: "https://example.com",
				S3Client:       &MockS3Client{},
			},
			ExpectError: filedropper.ErrInvalidDestination,
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
