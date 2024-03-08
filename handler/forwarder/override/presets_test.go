package override_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/override"
)

func TestPresets(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Presets []string
		Expect  []VerifyApply
	}{
		{
			Presets: []string{"aws/v1"},
			Expect: []VerifyApply{
				{
					Input: &s3.CopyObjectInput{
						CopySource:      aws.String("test-bucket/AWSLogs/723346149663/Config/us-west-2/2023/10/10/OversizedChangeNotification/AWS::SSM::ManagedInstanceInventory/i-0c08bc770c167d93c/723346149663_Config_us-west-2_ChangeNotification_AWS::SSM::ManagedInstanceInventory_i-0c08bc770c167d93c_20231010T203453Z_1696970093120.json.gz"),
						ContentEncoding: aws.String("gzip"),
					},
					Expect: &s3.CopyObjectInput{
						CopySource:        aws.String("test-bucket/AWSLogs/723346149663/Config/us-west-2/2023/10/10/OversizedChangeNotification/AWS::SSM::ManagedInstanceInventory/i-0c08bc770c167d93c/723346149663_Config_us-west-2_ChangeNotification_AWS::SSM::ManagedInstanceInventory_i-0c08bc770c167d93c_20231010T203453Z_1696970093120.json.gz"),
						ContentType:       aws.String("application/x-aws-change"),
						ContentEncoding:   aws.String("gzip"),
						MetadataDirective: types.MetadataDirectiveReplace,
					},
				},
				{
					Input: &s3.CopyObjectInput{
						CopySource: aws.String("test-bucket/cloudwatchlogs/us-west-2/2024/02/27/22/quality-bird-logwriter-1-2024-02-27-22-16-04-7828720f-2bd1-4b15-9f4c-b33f06f4a9c0"),
					},
					Expect: &s3.CopyObjectInput{
						CopySource:        aws.String("test-bucket/cloudwatchlogs/us-west-2/2024/02/27/22/quality-bird-logwriter-1-2024-02-27-22-16-04-7828720f-2bd1-4b15-9f4c-b33f06f4a9c0"),
						ContentType:       aws.String("application/x-aws-cloudwatchlogs"),
						ContentEncoding:   aws.String("gzip"),
						MetadataDirective: types.MetadataDirectiveReplace,
					},
				},
				{
					Input: &s3.CopyObjectInput{
						CopySource:      aws.String("test-bucket/AWSLogs/123456789012/CloudTrail/us-west-2/2024/03/07/123456789012_CloudTrail_us-west-2_20240307T1735Z_avVctZJaEJudp7oI.json.gz"),
						ContentEncoding: aws.String("gzip"),
					},
					Expect: &s3.CopyObjectInput{
						CopySource:        aws.String("test-bucket/AWSLogs/123456789012/CloudTrail/us-west-2/2024/03/07/123456789012_CloudTrail_us-west-2_20240307T1735Z_avVctZJaEJudp7oI.json.gz"),
						ContentType:       aws.String("application/x-aws-cloudtrail"),
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
			ss, err := override.LoadPresets(logr.Discard(), tc.Presets...)
			if err != nil {
				t.Fatal(err)
			}

			set := override.Sets(ss)
			for i, pair := range tc.Expect {
				set.Apply(context.Background(), pair.Input)
				if diff := cmp.Diff(pair.Input, pair.Expect, cmpopts.IgnoreUnexported(s3.CopyObjectInput{})); diff != "" {
					t.Fatalf("failed to process %d: %s", i, diff)
				}
			}
		})
	}
}
