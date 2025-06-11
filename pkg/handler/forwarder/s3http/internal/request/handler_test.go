package request_test

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http/internal/request"
)

func TestHandler(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := r.Body.Close(); err != nil {
				t.Errorf("failed to close request body: %v", err)
			}
		}()

		body := r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				panic(err)
			}
			body = gr
		}

		if data, err := io.ReadAll(body); err != nil {
			w.WriteHeader(400)
		} else if string(data) != "test" {
			w.WriteHeader(400)
		}
	}))

	h := &request.Handler{
		URL:    s.URL,
		Client: s.Client(),
	}

	if err := h.Handle(context.Background(), strings.NewReader("test")); err != nil {
		t.Fatal("failed to handle request:", err)
	}

	h.GzipLevel = -1
	if err := h.Handle(context.Background(), strings.NewReader("test")); err != nil {
		t.Fatal("failed to handle gzipped request:", err)
	}
}
