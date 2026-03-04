package tagger

import (
	"testing"
)

func TestLookupNamespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		namespace    string
		expectFound  bool
		expectType   string
		expectDimKey string
	}{
		{"AWS/EC2", true, "ec2:instance", "InstanceId"},
		{"AWS/Lambda", true, "lambda:function", "FunctionName"},
		{"AWS/RDS", true, "rds:db", "DBInstanceIdentifier"},
		{"AWS/S3", true, "s3", "BucketName"},
		{"AWS/DynamoDB", true, "dynamodb:table", "TableName"},
		{"AWS/Unknown", false, "", ""},
		{"CustomNamespace", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.namespace, func(t *testing.T) {
			t.Parallel()
			m, ok := LookupNamespace(tt.namespace)
			if ok != tt.expectFound {
				t.Errorf("LookupNamespace(%q) found=%v, want %v", tt.namespace, ok, tt.expectFound)
			}
			if ok {
				if m.ResourceType != tt.expectType {
					t.Errorf("ResourceType=%q, want %q", m.ResourceType, tt.expectType)
				}
				if m.DimensionKey != tt.expectDimKey {
					t.Errorf("DimensionKey=%q, want %q", m.DimensionKey, tt.expectDimKey)
				}
			}
		})
	}
}

func TestAllResourceTypes(t *testing.T) {
	t.Parallel()

	types := AllResourceTypes()
	if len(types) == 0 {
		t.Fatal("AllResourceTypes returned empty slice")
	}

	seen := make(map[string]int)
	for _, rt := range types {
		seen[rt]++
		if seen[rt] > 1 {
			t.Errorf("duplicate resource type: %s", rt)
		}
	}
}

func TestExtractResourceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arn    string
		expect string
	}{
		{"arn:aws:ec2:us-east-1:123456789012:instance/i-0abcd1234", "i-0abcd1234"},
		{"arn:aws:lambda:us-east-1:123456789012:function:my-func", "my-func"},
		{"arn:aws:s3:::my-bucket", "my-bucket"},
		{"simple-id", "simple-id"},
	}

	for _, tt := range tests {
		t.Run(tt.arn, func(t *testing.T) {
			t.Parallel()
			got := ExtractResourceID(tt.arn)
			if got != tt.expect {
				t.Errorf("ExtractResourceID(%q) = %q, want %q", tt.arn, got, tt.expect)
			}
		})
	}
}
