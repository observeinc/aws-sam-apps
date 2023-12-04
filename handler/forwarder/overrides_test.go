package forwarder_test

import (
	"fmt"
	"testing"

	"github.com/observeinc/aws-sam-testing/handler/forwarder"

	"github.com/google/go-cmp/cmp"
)

func TestContentTypeOverridesErrors(t *testing.T) {
	t.Parallel()

	testcases := []string{
		"nonono",
		"\\=",
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			_, err := forwarder.NewContentTypeOverrides([]string{tc}, "=")
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestContentTypeOverrides(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		ContentTypeOverrides []string
		Delimiter            string
		Expect               map[string]string
	}{
		{
			ContentTypeOverrides: []string{
				".*=application/json",
				"",
			},
			Delimiter: "=",
			Expect: map[string]string{
				"s3://bucket-example/key.json": "application/json",
				"s3://another-example/0000":    "application/json",
			},
		},
		{
			ContentTypeOverrides: []string{
				"txt$!text/plain",
				".*!application/json",
			},
			Delimiter: "!",
			Expect: map[string]string{
				"s3://bucket-example/key.json":  "application/json",
				"s3://bucket-example/key.txt":   "text/plain",
				"s3://another-example/txt.0000": "application/json",
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			m, err := forwarder.NewContentTypeOverrides(tc.ContentTypeOverrides, tc.Delimiter)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tc.Expect {
				if diff := cmp.Diff(m.Match(k), v); diff != "" {
					t.Fatalf("failed to process %q: %s", k, diff)
				}
			}
		})
	}
}
