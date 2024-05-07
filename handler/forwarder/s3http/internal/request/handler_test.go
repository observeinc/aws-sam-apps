package request_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/s3http/internal/request"
)

func TestHandler(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))

	h := &request.Handler{
		URL:    s.URL,
		Client: s.Client(),
	}

	if err := h.Handle(context.Background(), strings.NewReader("test")); err != nil {
		t.Fatal("failed to handle request:", err)
	}
}
