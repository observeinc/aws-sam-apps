package handler

import "testing"

func TestParseS3URI(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		wantBucket string
		wantKey    string
		wantErr    bool
	}{
		{
			name:       "valid URI",
			uri:        "s3://observeinc/cloudwatchmetrics/filters/recommended.yaml",
			wantBucket: "observeinc",
			wantKey:    "cloudwatchmetrics/filters/recommended.yaml",
		},
		{
			name:       "valid URI with simple key",
			uri:        "s3://mybucket/mykey.yaml",
			wantBucket: "mybucket",
			wantKey:    "mykey.yaml",
		},
		{
			name:    "missing s3 prefix",
			uri:     "https://bucket/key",
			wantErr: true,
		},
		{
			name:    "no key",
			uri:     "s3://bucket",
			wantErr: true,
		},
		{
			name:    "empty key",
			uri:     "s3://bucket/",
			wantErr: true,
		},
		{
			name:    "empty string",
			uri:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := ParseS3URI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if bucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", bucket, tt.wantBucket)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}
