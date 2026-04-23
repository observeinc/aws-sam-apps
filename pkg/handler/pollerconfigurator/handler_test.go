package pollerconfigurator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
)

// mockRoundTripper intercepts HTTP requests so tests can inspect outgoing
// GQL mutations and return canned responses without hitting a real server.
type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

// testPollerConfig is the JSON config served by the config httptest server.
const testPollerConfig = `{
	"name": "test-poller",
	"datastreamId": "ds-999",
	"interval": "5m",
	"period": 300,
	"delay": 300,
	"queries": [{"namespace": "AWS/EC2", "metricNames": ["CPUUtilization"]}]
}`

// newConfigServer returns an httptest server that serves a valid poller config.
func newConfigServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testPollerConfig))
	}))
}

// cfnCapture tracks the CloudFormation callback response.
type cfnCapture struct {
	Response CfResponse
	server   *httptest.Server
}

func newCfnServer(t *testing.T) *cfnCapture {
	t.Helper()
	c := &cfnCapture{}
	c.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		_ = json.NewDecoder(r.Body).Decode(&c.Response)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(c.server.Close)
	return c
}

func newTestHandler(cfnURL, configURL string) Handler {
	return Handler{
		Logger:            logr.Discard(),
		ObserveAccountID:  "111222333",
		ObserveDomainName: "observeinc.com",
		WorkspaceID:       "ws-42",
		Region:            "us-east-1",
		AWSAccountID:      "999888777",
		ExternalRoleName:  "observe-role",
		PollerConfigURI:   configURL,
	}
}

func newTestRequest(cfnURL string) *Request {
	return &Request{
		RequestType:       "Create",
		ResponseURL:       cfnURL,
		StackId:           "arn:aws:cloudformation:us-east-1:123:stack/test/guid",
		RequestId:         "req-1",
		LogicalResourceId: "PollerCustomResource",
	}
}

func TestHandleCreate(t *testing.T) {
	configServer := newConfigServer(t)
	defer configServer.Close()

	cfn := newCfnServer(t)
	h := newTestHandler(cfn.server.URL, configServer.URL)

	var capturedQuery string
	gql := &gqlClient{
		httpClient: &http.Client{Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(req.Body)
				var gqlReq graphQLRequest
				_ = json.Unmarshal(body, &gqlReq)
				capturedQuery = gqlReq.Query
				resp := `{"data":{"createPoller":{"id":"poller-abc","name":"test"}}}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(resp)),
				}, nil
			},
		}},
		observeAccountID:  h.ObserveAccountID,
		observeDomainName: h.ObserveDomainName,
		logger:            logr.Discard(),
	}

	token := "test-token"
	req := newTestRequest(cfn.server.URL)
	req.RequestType = "Create"
	assumeRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", h.AWSAccountID, h.ExternalRoleName)

	_, err := h.handleCreate(context.Background(), req, gql, &token, assumeRoleArn)
	if err != nil {
		t.Fatalf("handleCreate() error = %v", err)
	}

	for _, want := range []string{
		`createPoller`,
		`workspaceId: "ws-42"`,
		fmt.Sprintf(`name: "test-poller-%s-%s"`, h.AWSAccountID, h.Region),
		`datastreamId: "ds-999"`,
		`region: "us-east-1"`,
		`assumeRoleArn: "arn:aws:iam::999888777:role/observe-role"`,
		`namespace: "AWS/EC2"`,
		`metricNames: ["CPUUtilization"]`,
	} {
		if !strings.Contains(capturedQuery, want) {
			t.Errorf("createPoller mutation missing %q\ngot: %s", want, capturedQuery)
		}
	}

	if cfn.Response.Status != "SUCCESS" {
		t.Errorf("CFN status = %q, want SUCCESS", cfn.Response.Status)
	}
	if cfn.Response.PhysicalResourceId != "poller-abc" {
		t.Errorf("CFN PhysicalResourceId = %q, want %q", cfn.Response.PhysicalResourceId, "poller-abc")
	}
}

func TestHandleUpdate(t *testing.T) {
	configServer := newConfigServer(t)
	defer configServer.Close()

	cfn := newCfnServer(t)
	h := newTestHandler(cfn.server.URL, configServer.URL)

	var capturedQuery string
	gql := &gqlClient{
		httpClient: &http.Client{Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(req.Body)
				var gqlReq graphQLRequest
				_ = json.Unmarshal(body, &gqlReq)
				capturedQuery = gqlReq.Query
				resp := `{"data":{"updatePoller":{"id":"existing-poller-id","name":"updated"}}}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(resp)),
				}, nil
			},
		}},
		observeAccountID:  h.ObserveAccountID,
		observeDomainName: h.ObserveDomainName,
		logger:            logr.Discard(),
	}

	token := "test-token"
	req := newTestRequest(cfn.server.URL)
	req.RequestType = "Update"
	req.PhysicalResourceId = "existing-poller-id"
	assumeRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", h.AWSAccountID, h.ExternalRoleName)

	_, err := h.handleUpdate(context.Background(), req, gql, &token, assumeRoleArn)
	if err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}

	for _, want := range []string{
		`updatePoller`,
		`id: "existing-poller-id"`,
		fmt.Sprintf(`name: "test-poller-%s-%s"`, h.AWSAccountID, h.Region),
		`datastreamId: "ds-999"`,
		`region: "us-east-1"`,
	} {
		if !strings.Contains(capturedQuery, want) {
			t.Errorf("updatePoller mutation missing %q\ngot: %s", want, capturedQuery)
		}
	}

	if cfn.Response.Status != "SUCCESS" {
		t.Errorf("CFN status = %q, want SUCCESS", cfn.Response.Status)
	}
	if cfn.Response.PhysicalResourceId != "existing-poller-id" {
		t.Errorf("CFN PhysicalResourceId = %q, want %q", cfn.Response.PhysicalResourceId, "existing-poller-id")
	}
}

func TestHandleUpdate_MissingPhysicalResourceId(t *testing.T) {
	cfn := newCfnServer(t)
	h := newTestHandler(cfn.server.URL, "http://unused")

	gqlCalled := false
	gql := &gqlClient{
		httpClient: &http.Client{Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				gqlCalled = true
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))}, nil
			},
		}},
		logger: logr.Discard(),
	}

	token := "test-token"
	req := newTestRequest(cfn.server.URL)
	req.RequestType = "Update"
	req.PhysicalResourceId = ""

	_, err := h.handleUpdate(context.Background(), req, gql, &token, "arn:unused")
	if err == nil {
		t.Fatal("handleUpdate() should error when PhysicalResourceId is empty")
	}

	if gqlCalled {
		t.Error("GQL should not be called when PhysicalResourceId is missing")
	}
	if cfn.Response.Status != "FAILED" {
		t.Errorf("CFN status = %q, want FAILED", cfn.Response.Status)
	}
}

func TestHandleDelete(t *testing.T) {
	cfn := newCfnServer(t)
	h := newTestHandler(cfn.server.URL, "http://unused")

	var capturedQuery string
	gql := &gqlClient{
		httpClient: &http.Client{Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(req.Body)
				var gqlReq graphQLRequest
				_ = json.Unmarshal(body, &gqlReq)
				capturedQuery = gqlReq.Query
				resp := `{"data":{"deletePoller":{"success":true}}}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(resp)),
				}, nil
			},
		}},
		observeAccountID:  h.ObserveAccountID,
		observeDomainName: h.ObserveDomainName,
		logger:            logr.Discard(),
	}

	token := "test-token"
	req := newTestRequest(cfn.server.URL)
	req.RequestType = "Delete"
	req.PhysicalResourceId = "poller-to-delete"

	_, err := h.handleDelete(req, gql, &token)
	if err != nil {
		t.Fatalf("handleDelete() error = %v", err)
	}

	for _, want := range []string{
		`deletePoller`,
		`id: "poller-to-delete"`,
	} {
		if !strings.Contains(capturedQuery, want) {
			t.Errorf("deletePoller mutation missing %q\ngot: %s", want, capturedQuery)
		}
	}

	if cfn.Response.Status != "SUCCESS" {
		t.Errorf("CFN status = %q, want SUCCESS", cfn.Response.Status)
	}
}

func TestHandleDelete_NoPhysicalResourceId(t *testing.T) {
	cfn := newCfnServer(t)
	h := newTestHandler(cfn.server.URL, "http://unused")

	gqlCalled := false
	gql := &gqlClient{
		httpClient: &http.Client{Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				gqlCalled = true
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))}, nil
			},
		}},
		logger: logr.Discard(),
	}

	token := "test-token"
	req := newTestRequest(cfn.server.URL)
	req.RequestType = "Delete"
	req.PhysicalResourceId = ""

	_, err := h.handleDelete(req, gql, &token)
	if err != nil {
		t.Fatalf("handleDelete() error = %v", err)
	}

	if gqlCalled {
		t.Error("GQL should not be called when deleting with no PhysicalResourceId")
	}
	if cfn.Response.Status != "SUCCESS" {
		t.Errorf("CFN status = %q, want SUCCESS (no-op delete)", cfn.Response.Status)
	}
}

func TestHandleDelete_GQLError_StillReportsSuccess(t *testing.T) {
	cfn := newCfnServer(t)
	h := newTestHandler(cfn.server.URL, "http://unused")

	gql := &gqlClient{
		httpClient: &http.Client{Transport: &mockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewBufferString("internal error")),
				}, nil
			},
		}},
		observeAccountID:  h.ObserveAccountID,
		observeDomainName: h.ObserveDomainName,
		logger:            logr.Discard(),
	}

	token := "test-token"
	req := newTestRequest(cfn.server.URL)
	req.RequestType = "Delete"
	req.PhysicalResourceId = "poller-xyz"

	_, err := h.handleDelete(req, gql, &token)
	if err != nil {
		t.Fatalf("handleDelete() error = %v", err)
	}

	if cfn.Response.Status != "SUCCESS" {
		t.Errorf("CFN status = %q, want SUCCESS even when GQL fails", cfn.Response.Status)
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
