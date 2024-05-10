package decoders_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/s3http/internal/decoders"
)

var update = flag.Bool("update-golden-files", false, "Instruct the test to write golden files")

func TestDecoders(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		ContentType     string
		ContentEncoding string
		InputFile       string
	}{
		{
			ContentType: "application/json",
			InputFile:   "testdata/example.json",
		},
		{
			ContentType: "application/x-ndjson",
			InputFile:   "testdata/example.ndjson",
		},
		{
			ContentType: "application/x-aws-config",
			InputFile:   "testdata/config.json",
		},
		{
			ContentType: "application/x-aws-cloudtrail",
			InputFile:   "testdata/cloudtrail.json",
		},
		{
			ContentType: "text/csv",
			InputFile:   "testdata/example.csv",
		},
		{
			ContentType:     "application/x-aws-vpcflowlogs",
			ContentEncoding: "gzip",
			InputFile:       "testdata/vpcflowlogs.log.gz",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.InputFile, func(t *testing.T) {
			t.Parallel()

			fn, err := decoders.Get(tt.ContentEncoding, tt.ContentType)
			if err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer

			enc := json.NewEncoder(&buf)
			for dec := fn(readFile(t, tt.InputFile)); dec.More(); {
				var v json.RawMessage
				if err := dec.Decode(&v); err != nil {
					t.Fatal(err)
				}
				if err := enc.Encode(v); err != nil {
					t.Fatal(err)
				}
			}

			if *update {
				t.Log("overwriting file")
				writeFile(t, tt.InputFile+".golden", buf.Bytes())
			}

			compareFile(t, tt.InputFile+".golden", &buf)
		})
	}
}

func readFile(t *testing.T, filename string) io.Reader {
	t.Helper()
	file, err := os.Open(filename)
	if err != nil {
		t.Fatal("failed to open file:", err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}

func writeFile(t *testing.T, filename string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filename, data, 0o666); err != nil {
		t.Fatal("failed to write file:", err)
	}
}

func compareFile(t *testing.T, filename string, contents io.Reader) {
	t.Helper()
	var a, b bytes.Buffer

	if _, err := a.ReadFrom(readFile(t, filename)); err != nil {
		t.Fatal("failed to read file:", err)
	}

	if _, err := b.ReadFrom(contents); err != nil {
		t.Fatal("failed to read file:", err)
	}

	t.Log(a)
	t.Log(b)
	if diff := cmp.Diff(a.String(), b.String()); diff != "" {
		t.Fatal(diff)
	}
}
