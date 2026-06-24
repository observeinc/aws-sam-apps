package pollerconfigurator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestDownloadPollerConfig(t *testing.T) {
	tests := []struct {
		name                   string
		body                   string
		statusCode             int
		wantErr                bool
		errSubstr              string
		wantName               string
		wantPeriod             int64
		wantAttachResourceTags *bool
	}{
		{
			name:       "valid config",
			body:       `{"name":"test-poller","interval":"5m","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantName:   "test-poller",
			wantPeriod: 300,
		},
		{
			name:                   "attachResourceTags true",
			body:                   `{"name":"test-poller","interval":"5m","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}],"attachResourceTags":true}`,
			statusCode:             http.StatusOK,
			wantName:               "test-poller",
			wantPeriod:             300,
			wantAttachResourceTags: boolPtr(true),
		},
		{
			name:                   "attachResourceTags false",
			body:                   `{"name":"test-poller","interval":"5m","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}],"attachResourceTags":false}`,
			statusCode:             http.StatusOK,
			wantName:               "test-poller",
			wantPeriod:             300,
			wantAttachResourceTags: boolPtr(false),
		},
		{
			name:       "attachResourceTags absent",
			body:       `{"name":"test-poller","interval":"5m","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantName:   "test-poller",
			wantPeriod: 300,
			// wantAttachResourceTags nil — field should be absent from parsed config
		},
		{
			name:       "missing queries",
			body:       `{"name":"test-poller","interval":"5m","period":300,"delay":300,"queries":[]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "at least one query",
		},
		{
			name:       "zero period",
			body:       `{"name":"test-poller","interval":"5m","period":0,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "period must be positive",
		},
		{
			name:       "zero delay",
			body:       `{"name":"test-poller","interval":"5m","period":300,"delay":0,"queries":[{"namespace":"AWS/EC2"}]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
			errSubstr:  "delay must be positive",
		},
		{
			name:       "missing interval",
			body:       `{"name":"test-poller","period":300,"delay":300,"queries":[{"namespace":"AWS/EC2"}]}`,
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
				_, _ = w.Write([]byte(tt.body))
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
				switch {
				case tt.wantAttachResourceTags == nil && cfg.AttachResourceTags != nil:
					t.Errorf("cfg.AttachResourceTags = %v, want nil", *cfg.AttachResourceTags)
				case tt.wantAttachResourceTags != nil && cfg.AttachResourceTags == nil:
					t.Errorf("cfg.AttachResourceTags = nil, want %v", *tt.wantAttachResourceTags)
				case tt.wantAttachResourceTags != nil && cfg.AttachResourceTags != nil && *cfg.AttachResourceTags != *tt.wantAttachResourceTags:
					t.Errorf("cfg.AttachResourceTags = %v, want %v", *cfg.AttachResourceTags, *tt.wantAttachResourceTags)
				}
			}
		})
	}
}
