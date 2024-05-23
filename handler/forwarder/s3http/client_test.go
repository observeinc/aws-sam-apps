package s3http_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-cmp/cmp"
	"github.com/lithammer/dedent"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/s3http"
	"github.com/observeinc/aws-sam-apps/handler/handlertest"
)

func format(s string) string {
	return strings.TrimLeft(dedent.Dedent(s), "\n")
}

func TestClientPut(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		*s3.PutObjectInput
		Path   string
		Expect string
	}{
		{
			PutObjectInput: &s3.PutObjectInput{
				Bucket:      aws.String("test"),
				Key:         aws.String("example.txt"),
				ContentType: aws.String("text/plain"),
				Body:        strings.NewReader("hello world"),
			},
			Expect: format(`
				POST /?content-type=text%2Fplain&key=example.txt HTTP/1.1
				Host: 127.0.0.1:<removed>
				Accept-Encoding: gzip
				Content-Length: 23
				Content-Type: application/x-ndjson
				User-Agent: Go-http-client/1.1

				{"text":"hello world"}
			`),
		},
		{
			PutObjectInput: &s3.PutObjectInput{
				Bucket:      aws.String("test"),
				Key:         aws.String("deeply/nested/example.json"),
				ContentType: aws.String("application/json"),
				Body:        strings.NewReader(`{"hello": "world"}`),
			},
			Path: "v1/http",
			Expect: format(`
					POST /v1/http?content-type=application%2Fjson&key=deeply%2Fnested%2Fexample.json HTTP/1.1
					Host: 127.0.0.1:<removed>
					Accept-Encoding: gzip
					Content-Length: 19
					Content-Type: application/x-ndjson
					User-Agent: Go-http-client/1.1

					{"hello": "world"}
				`),
		},
	}

	for i, tt := range testcases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			var (
				mu  sync.Mutex
				buf bytes.Buffer
			)

			// Our server will just dump requests to buffer
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				// port number changes on every run, must be stripped
				r.Host = strings.TrimRight(r.Host, "0123456789") + "<removed>"
				d, _ := httputil.DumpRequest(r, true)
				if _, err := buf.Write(bytes.ReplaceAll(d, []byte("\r"), nil)); err != nil {
					w.WriteHeader(400)
				}
			}))

			// upload object
			client, err := s3http.New(&s3http.Config{
				DestinationURI:     fmt.Sprintf("%s/%s", s.URL, tt.Path),
				GetObjectAPIClient: &handlertest.S3Client{},
				RequestGzipLevel:   aws.Int(0),
				HTTPClient:         s.Client(),
			})
			if err != nil {
				t.Fatal(err)
			}

			_, err = client.PutObject(context.Background(), tt.PutObjectInput)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("received:\n%s", buf.String())
			if diff := cmp.Diff(buf.String(), tt.Expect); diff != "" {
				t.Fatal("response does not match", diff)
			}
		})
	}
}
