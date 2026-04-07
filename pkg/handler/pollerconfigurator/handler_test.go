package pollerconfigurator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
)

func TestParsePayload(t *testing.T) {
	cfnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cfnServer.Close()

	h := Handler{Logger: logr.Discard()}

	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: []byte(fmt.Sprintf(`{
				"RequestType": "Create",
				"ResponseURL": %q,
				"StackId": "arn:aws:cloudformation:us-east-1:123:stack/test/guid",
				"RequestId": "req-1",
				"ResourceType": "Custom::PollerConfigurator",
				"LogicalResourceId": "PollerCustomResource"
			}`, cfnServer.URL)),
		},
		{
			name:    "invalid JSON",
			payload: []byte(`{broken`),
			wantErr: true,
		},
		{
			name:    "empty payload",
			payload: []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := h.parsePayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && req.RequestType != "Create" {
				t.Errorf("parsePayload() RequestType = %q, want %q", req.RequestType, "Create")
			}
		})
	}
}

func TestReportStatusWithPhysicalID(t *testing.T) {
	var receivedBody CfResponse

	cfnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer cfnServer.Close()

	h := Handler{Logger: logr.Discard()}

	req := Request{
		StackId:           "arn:aws:cloudformation:us-east-1:123:stack/test/guid",
		RequestId:         "req-1",
		LogicalResourceId: "PollerCustomResource",
		ResponseURL:       cfnServer.URL,
	}

	err := h.reportStatusWithPhysicalID(req, true, "poller created", "poller-12345")
	if err != nil {
		t.Fatalf("reportStatusWithPhysicalID() error = %v", err)
	}

	if receivedBody.Status != "SUCCESS" {
		t.Errorf("Status = %q, want %q", receivedBody.Status, "SUCCESS")
	}
	if receivedBody.PhysicalResourceId != "poller-12345" {
		t.Errorf("PhysicalResourceId = %q, want %q", receivedBody.PhysicalResourceId, "poller-12345")
	}
	if receivedBody.Reason != "poller created" {
		t.Errorf("Reason = %q, want %q", receivedBody.Reason, "poller created")
	}
}

func TestReportStatusFailure(t *testing.T) {
	var receivedBody CfResponse

	cfnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer cfnServer.Close()

	h := Handler{Logger: logr.Discard()}

	req := Request{
		StackId:           "arn:aws:cloudformation:us-east-1:123:stack/test/guid",
		RequestId:         "req-1",
		LogicalResourceId: "PollerCustomResource",
		ResponseURL:       cfnServer.URL,
	}

	err := h.reportStatusWithPhysicalID(req, false, "something failed", "")
	if err != nil {
		t.Fatalf("reportStatusWithPhysicalID() error = %v", err)
	}

	if receivedBody.Status != "FAILED" {
		t.Errorf("Status = %q, want %q", receivedBody.Status, "FAILED")
	}
	if receivedBody.PhysicalResourceId != "lambda-pollerconfigurator" {
		t.Errorf("PhysicalResourceId = %q, want fallback", receivedBody.PhysicalResourceId)
	}
}

func TestReportAndError(t *testing.T) {
	cfnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cfnServer.Close()

	h := Handler{Logger: logr.Discard()}

	req := &Request{
		ResponseURL:       cfnServer.URL,
		StackId:           "stack-1",
		RequestId:         "req-1",
		LogicalResourceId: "res-1",
	}

	err := h.reportAndError("test failure", req, fmt.Errorf("underlying error"))
	if err == nil {
		t.Fatal("reportAndError() should return error")
	}
	if !strings.Contains(err.Error(), "test failure") {
		t.Errorf("error should contain reason, got: %v", err)
	}
	if !strings.Contains(err.Error(), "underlying error") {
		t.Errorf("error should contain underlying error, got: %v", err)
	}
}

func TestReportAndError_NilError(t *testing.T) {
	cfnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cfnServer.Close()

	h := Handler{Logger: logr.Discard()}

	req := &Request{
		ResponseURL:       cfnServer.URL,
		StackId:           "stack-1",
		RequestId:         "req-1",
		LogicalResourceId: "res-1",
	}

	err := h.reportAndError("no underlying error", req, nil)
	if err == nil {
		t.Fatal("reportAndError() should return error")
	}
	if !strings.Contains(err.Error(), "no underlying error") {
		t.Errorf("error should contain reason, got: %v", err)
	}
}

func TestHandlerNew(t *testing.T) {
	cfg := &Config{
		ObserveAccountID:  "123",
		ObserveDomainName: "observeinc.com",
		SecretName:        "secret",
		PollerConfigURI:   "s3://bucket/config.json",
		ExternalRoleName:  "observe-role",
		WorkspaceID:       "456",
		Region:            "us-east-1",
		AWSAccountID:      "999888777",
	}

	h, err := New(cfg, logr.Discard())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if h.ObserveAccountID != "123" {
		t.Errorf("ObserveAccountID = %q, want %q", h.ObserveAccountID, "123")
	}
	if h.WorkspaceID != "456" {
		t.Errorf("WorkspaceID = %q, want %q", h.WorkspaceID, "456")
	}
	if h.AWSAccountID != "999888777" {
		t.Errorf("AWSAccountID = %q, want %q", h.AWSAccountID, "999888777")
	}
}
