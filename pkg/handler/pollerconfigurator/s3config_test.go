package pollerconfigurator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3URIToHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{
			name: "valid URI",
			uri:  "s3://my-bucket/path/to/config.json",
			want: "https://my-bucket.s3.amazonaws.com/path/to/config.json",
		},
		{
			name:    "missing s3 prefix",
			uri:     "https://my-bucket/config.json",
			wantErr: true,
		},
		{
			name:    "no key",
			uri:     "s3://my-bucket",
			wantErr: true,
		},
		{
			name:    "empty key",
			uri:     "s3://my-bucket/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s3URIToHTTPS(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("s3URIToHTTPS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("s3URIToHTTPS() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadPollerConfig(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		statusCode int
		wantErr    bool
		errSubstr  string
		wantName   string
		wantPeriod int64
	}{
		{
			name:       "valid config",
			body:       `{"name":"test-poller","datastreamId":"12345","interval":"5m","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantName:   "test-poller",
			wantPeriod: 300,
		},
		{
			name:       "missing queries",
			body:       `{"name":"test-poller","datastreamId":"12345","interval":"5m","period":300,"delay":300,"queries":[]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "at least one query",
		},
		{
			name:       "zero period",
			body:       `{"name":"test-poller","datastreamId":"12345","interval":"5m","period":0,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "period must be positive",
		},
		{
			name:       "zero delay",
			body:       `{"name":"test-poller","datastreamId":"12345","interval":"5m","period":300,"delay":0,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "delay must be positive",
		},
		{
			name:       "missing datastreamId",
			body:       `{"name":"test-poller","interval":"5m","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "must include datastreamId",
		},
		{
			name:       "missing interval",
			body:       `{"name":"test-poller","datastreamId":"12345","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "must include interval",
		},
		{
			name:       "invalid JSON",
			body:       `{invalid`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "parse poller config JSON",
		},
		{
			name:       "HTTP error",
			body:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errSubstr:  "HTTP 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			cfg, err := downloadPollerConfigFromURL(context.Background(), srv.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadPollerConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("downloadPollerConfig() error = %q, want substr %q", err.Error(), tt.errSubstr)
			}
			if !tt.wantErr {
				if cfg.Name != tt.wantName {
					t.Errorf("cfg.Name = %q, want %q", cfg.Name, tt.wantName)
				}
				if cfg.Period != tt.wantPeriod {
					t.Errorf("cfg.Period = %d, want %d", cfg.Period, tt.wantPeriod)
				}
			}
		})
	}
}
