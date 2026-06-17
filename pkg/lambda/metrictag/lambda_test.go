package metrictag

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/pkg/handler/metrictag/enrich"
)

func newTestLambda(t *testing.T) *Lambda {
	t.Helper()
	e, err := enrich.New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("enrich.New: %v", err)
	}
	l := &Lambda{
		Logger:   logr.Discard(),
		enricher: e,
	}
	l.Entrypoint = l.handle
	return l
}

const unknownLine = `{"namespace":"Custom/Unknown","region":"us-east-1","account_id":"123","metric_name":"M","dimensions":{},"timestamp":1,"value":{"count":1},"unit":"Count"}`

func TestHandle_singleRecord_trailingNewline(t *testing.T) {
	l := newTestLambda(t)
	ev := events.KinesisFirehoseEvent{
		Records: []events.KinesisFirehoseEventRecord{
			{RecordID: "r1", Data: []byte(unknownLine + "\n")},
		},
	}
	resp, err := l.handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(resp.Records))
	}
	rec := resp.Records[0]
	if rec.Result != events.KinesisFirehoseTransformedStateOk {
		t.Errorf("expected Ok, got %v", rec.Result)
	}
	if !bytes.HasSuffix(rec.Data, []byte("\n")) {
		t.Errorf("output should end with newline, got %q", rec.Data)
	}
}

func TestHandle_multiLineRecord(t *testing.T) {
	l := newTestLambda(t)
	record := strings.Join([]string{unknownLine, unknownLine, unknownLine}, "\n") + "\n"
	ev := events.KinesisFirehoseEvent{
		Records: []events.KinesisFirehoseEventRecord{
			{RecordID: "r1", Data: []byte(record)},
		},
	}
	resp, err := l.handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	rec := resp.Records[0]
	if rec.Result != events.KinesisFirehoseTransformedStateOk {
		t.Errorf("expected Ok, got %v", rec.Result)
	}
	lines := bytes.Split(bytes.TrimRight(rec.Data, "\n"), []byte("\n"))
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), rec.Data)
	}
	for i, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			t.Errorf("line %d not valid JSON: %v", i, err)
		}
		if obj["resource_tags"] == nil {
			t.Errorf("line %d missing resource_tags", i)
		}
	}
	if !bytes.HasSuffix(rec.Data, []byte("\n")) {
		t.Errorf("output should end with newline, got %q", rec.Data)
	}
}

func TestHandle_invalidJSON_processingFailed(t *testing.T) {
	l := newTestLambda(t)
	raw := []byte("not valid json\n")
	ev := events.KinesisFirehoseEvent{
		Records: []events.KinesisFirehoseEventRecord{
			{RecordID: "r1", Data: raw},
		},
	}
	resp, err := l.handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	rec := resp.Records[0]
	if rec.Result != events.KinesisFirehoseTransformedStateProcessingFailed {
		t.Errorf("expected ProcessingFailed, got %v", rec.Result)
	}
	if !bytes.Equal(rec.Data, raw) {
		t.Errorf("failed record should carry original data, got %q", rec.Data)
	}
}

func TestHandle_multipleRecords_independentResults(t *testing.T) {
	l := newTestLambda(t)
	ev := events.KinesisFirehoseEvent{
		Records: []events.KinesisFirehoseEventRecord{
			{RecordID: "ok", Data: []byte(unknownLine + "\n")},
			{RecordID: "bad", Data: []byte("not json\n")},
			{RecordID: "ok2", Data: []byte(unknownLine + "\n")},
		},
	}
	resp, err := l.handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(resp.Records))
	}
	states := map[string]string{
		"ok":  events.KinesisFirehoseTransformedStateOk,
		"bad": events.KinesisFirehoseTransformedStateProcessingFailed,
		"ok2": events.KinesisFirehoseTransformedStateOk,
	}
	for _, rec := range resp.Records {
		want := states[rec.RecordID]
		if rec.Result != want {
			t.Errorf("record %q: expected %v, got %v", rec.RecordID, want, rec.Result)
		}
	}
}

func TestHandle_emptyData_producesOkWithEmptyBody(t *testing.T) {
	l := newTestLambda(t)
	ev := events.KinesisFirehoseEvent{
		Records: []events.KinesisFirehoseEventRecord{
			{RecordID: "r1", Data: []byte("   \n  ")},
		},
	}
	resp, err := l.handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	rec := resp.Records[0]
	if rec.Result != events.KinesisFirehoseTransformedStateOk {
		t.Errorf("expected Ok for blank record, got %v", rec.Result)
	}
	if len(rec.Data) != 0 {
		t.Errorf("expected empty output for blank record, got %q", rec.Data)
	}
}
