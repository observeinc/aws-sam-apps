package seekable_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/seekable"
)

func TestFromReaderInMemory(t *testing.T) {
	t.Parallel()

	rs, cleanup, err := seekable.FromReader(strings.NewReader("hello"), 1024)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Fatal(err)
		}
	}()

	data, err := io.ReadAll(rs)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected payload: %q", string(data))
	}

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("expected seekable reader: %v", err)
	}
}

func TestFromReaderSpillsToDiskAndCleansUp(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte("a"), 4096)
	rs, cleanup, err := seekable.FromReader(bytes.NewBuffer(payload), 1024)
	if err != nil {
		t.Fatal(err)
	}

	file, ok := rs.(*os.File)
	if !ok {
		t.Fatalf("expected spilled *os.File, got %T", rs)
	}
	fileName := file.Name()

	got, err := io.ReadAll(rs)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("spooled payload mismatch")
	}

	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		t.Fatalf("expected temp file to be removed, stat err: %v", err)
	}
}
