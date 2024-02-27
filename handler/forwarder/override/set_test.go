package override_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gopkg.in/yaml.v3"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/override"
)

func ptr(s string) *string {
	return &s
}

type VerifyApply struct {
	Input  *s3.CopyObjectInput
	Expect *s3.CopyObjectInput
}

func TestSet(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		YAML        string
		ExpectError *regexp.Regexp
		Expect      []VerifyApply
	}{
		{
			YAML: trimLeadingWhitespace(`
			---
			rules:
			  - id: "test"
			  - id: "test"
			`),
			ExpectError: regexp.MustCompile(`rule "test": duplicate ID`),
		},

		{
			YAML: trimLeadingWhitespace(`
			---
			rules:
			  - match:
			      source: '\.gz$'
			      content-encoding: '^$'
			    override:
			      content-encoding: 'gzip'
			    continue: true
			  - match:
			      source: '\.json(\.gz)?'
			      content-type: '^$'
			    override:
			      content-type: 'application/x-ndjson'
			  - match:
			      source: '.*'
			      content-type: '^$'
			    override:
			      content-type: 'text/plain'
			`),
			Expect: []VerifyApply{
				{
					Input: &s3.CopyObjectInput{
						CopySource: aws.String("source/key.json.gz"),
					},
					Expect: &s3.CopyObjectInput{
						CopySource:        aws.String("source/key.json.gz"),
						ContentType:       aws.String("application/x-ndjson"),
						ContentEncoding:   aws.String("gzip"),
						MetadataDirective: types.MetadataDirectiveReplace,
					},
				},
				{
					Input: &s3.CopyObjectInput{
						CopySource:  aws.String("source/key.json.gz"),
						ContentType: aws.String("already/set"),
					},
					Expect: &s3.CopyObjectInput{
						CopySource:        aws.String("source/key.json.gz"),
						ContentType:       aws.String("already/set"),
						ContentEncoding:   aws.String("gzip"),
						MetadataDirective: types.MetadataDirectiveReplace,
					},
				},
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			var set override.Set
			err := yaml.Unmarshal([]byte(tc.YAML), &set)
			if err == nil {
				err = set.Validate()
			}
			switch {
			case err == nil && tc.ExpectError == nil:
				// ok
			case err == nil && tc.ExpectError != nil:
				t.Fatal("expected error")
			case err != nil && tc.ExpectError == nil:
				t.Fatal("unexpected error:", err)
			case !tc.ExpectError.MatchString(err.Error()):
				t.Fatal("error does not match expected:", err)
			}

			for i, pair := range tc.Expect {
				set.Apply(context.Background(), pair.Input)
				if diff := cmp.Diff(pair.Input, pair.Expect, cmpopts.IgnoreUnexported(s3.CopyObjectInput{})); diff != "" {
					t.Fatalf("failed to process %d: %s", i, diff)
				}
			}
		})
	}
}
