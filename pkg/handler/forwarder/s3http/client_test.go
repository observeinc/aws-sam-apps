package s3http_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func format(s string) string {
	return strings.TrimLeft(dedent.Dedent(s), "\n")
}

func TestClientPut(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		*s3.PutObjectInput
		Path                string
		BatchFormat         string
		AuthorizationHeader string
		Expect              string
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
				Key:         aws.String("config.json"),
				ContentType: aws.String("text/plain"),
				Body:        strings.NewReader("hello world"),
			},
			BatchFormat:         s3http.BatchFormatJSONArray,
			AuthorizationHeader: "Basic dXNlcjpwYXNz",
			Expect: strings.TrimSuffix(format(`
				POST /?content-type=text%2Fplain&key=config.json HTTP/1.1
				Host: 127.0.0.1:<removed>
				Accept-Encoding: gzip
				Authorization: Basic dXNlcjpwYXNz
				Content-Length: 24
				Content-Type: application/json
				User-Agent: Go-http-client/1.1

				[{"text":"hello world"}]
			`), "\n"),
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
				DestinationURI:      fmt.Sprintf("%s/%s", s.URL, tt.Path),
				GetObjectAPIClient:  &awstest.S3Client{},
				HTTPClient:          s.Client(),
				BatchFormat:         tt.BatchFormat,
				AuthorizationHeader: tt.AuthorizationHeader,
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

func TestCopyConfigSnapshotAsJSONArray(t *testing.T) {
	t.Parallel()

	const snapshot = `{"fileVersion":"1.0","configurationItems":[{"awsAccountId":"123456789012","ARN":"arn:aws:ec2:us-west-2:123456789012:instance/i-1","resourceType":"AWS::EC2::Instance"},{"awsAccountId":"123456789012","ARN":"arn:aws:s3:::example","resourceType":"AWS::S3::Bucket"}]}`
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := io.WriteString(writer, snapshot); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	var body []byte
	server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		var err error
		body, err = io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected content type %q", got)
		}
	}))
	defer server.Close()

	client, err := s3http.New(&s3http.Config{
		DestinationURI: server.URL,
		GetObjectAPIClient: &awstest.S3Client{GetObjectFunc: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(compressed.Bytes()))}, nil
		}},
		HTTPClient:  server.Client(),
		BatchFormat: s3http.BatchFormatJSONArray,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:          aws.String("destination"),
		Key:             aws.String("snapshot.json.gz"),
		CopySource:      aws.String("config-bucket/snapshot.json.gz"),
		ContentType:     aws.String("application/x-aws-config"),
		ContentEncoding: aws.String("gzip"),
	})
	if err != nil {
		t.Fatal(err)
	}

	var items []map[string]any
	if err := json.Unmarshal(body, &items); err != nil {
		t.Fatalf("request body is not a JSON array: %v; body: %s", err, body)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if got := items[0]["awsAccountId"]; got != "123456789012" {
		t.Errorf("unexpected account ID %#v", got)
	}
}

// TestCopyObjectGzipInference verifies that when a custom content-type override sets
// "application/x-aws-elasticloadbalancing" but leaves content-encoding nil, CopyObject
// still correctly infers "gzip" from the ".gz" key suffix and decompresses the body.
func TestCopyObjectGzipInference(t *testing.T) {
	t.Parallel()

	// Build a gzip-compressed body containing two SSV lines:
	// line 1: header (space-separated field names)
	// line 2: a single ALB-like log record
	const (
		header = "type time elb client:port target:port request_processing_time"
		record = `https 2024-01-01T00:00:00.000000Z my-alb 1.2.3.4:1234 10.0.0.1:80 0.001`
	)

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	if _, err := io.WriteString(gw, header+"\n"+record+"\n"); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	gzBytes := gzBuf.Bytes()

	// Capture requests sent to the HTTP destination.
	var (
		mu      sync.Mutex
		reqBody bytes.Buffer
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		reqBody.Write(body)
	}))
	defer srv.Close()

	// Mock S3 client: returns our gzip body with content-type set but NO content-encoding.
	mockS3 := &awstest.S3Client{
		GetObjectFunc: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			size := int64(len(gzBytes))
			return &s3.GetObjectOutput{
				Body:            io.NopCloser(bytes.NewReader(gzBytes)),
				ContentLength:   &size,
				ContentType:     aws.String("application/x-aws-elasticloadbalancing"),
				ContentEncoding: nil, // simulates override setting type but not encoding
			}, nil
		},
	}

	client, err := s3http.New(&s3http.Config{
		DestinationURI:     srv.URL,
		GetObjectAPIClient: mockS3,
		HTTPClient:         srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:          aws.String("dst-bucket"),
		Key:             aws.String("output/alb.log"),
		CopySource:      aws.String("src-bucket/logs/alb-2024-01-01.log.gz"),
		ContentType:     aws.String("application/x-aws-elasticloadbalancing"),
		ContentEncoding: nil,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := reqBody.String()
	t.Logf("received body:\n%s", got)

	// The decoded output should contain the SSV-decoded JSON record, not raw binary.
	// Verify no raw gzip magic bytes leaked through.
	if strings.Contains(got, "\x1f\x8b") {
		t.Fatal("response body contains raw gzip magic bytes — gzip was not decompressed")
	}

	// Verify the decoded record fields appear as JSON keys/values.
	for _, want := range []string{`"type"`, `"https"`, `"elb"`, `"my-alb"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in decoded output, got:\n%s", want, got)
		}
	}

	// Ensure it's valid ndjson (each line should be a JSON object).
	for _, line := range strings.Split(strings.TrimSpace(got), "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
			t.Errorf("line is not a JSON object: %q", line)
		}
	}
}
