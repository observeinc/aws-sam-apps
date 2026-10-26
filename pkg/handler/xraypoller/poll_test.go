package xraypoller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/go-logr/logr"
)

func TestExcludeService(t *testing.T) {
	cases := []struct {
		filter, service, want string
	}{
		{"", "", ""},
		{DefaultAPIGatewayFilter, "", DefaultAPIGatewayFilter},
		{"", "poller", `!service("poller")`},
		{DefaultAPIGatewayFilter, "poller", `(` + DefaultAPIGatewayFilter + `) AND !service("poller")`},
	}
	for _, c := range cases {
		if got := ExcludeService(c.filter, c.service); got != c.want {
			t.Errorf("ExcludeService(%q, %q) = %q, want %q", c.filter, c.service, got, c.want)
		}
	}
}

func TestWindow(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	lag := 90 * time.Second
	lookback := 60 * time.Second
	maxWindow := 15 * time.Minute

	start, end, ok := window(now, time.Time{}, false, lag, lookback, maxWindow)
	if !ok {
		t.Fatal("expected a window")
	}
	if !end.Equal(now.Add(-lag)) || !start.Equal(end.Add(-lookback)) {
		t.Fatalf("first window %s .. %s", start, end)
	}

	cursor := end
	start, end, ok = window(now.Add(time.Minute), cursor, true, lag, lookback, maxWindow)
	if !ok || !start.Equal(cursor) {
		t.Fatalf("cursor window start %s ok %v", start, ok)
	}
	if end.Sub(start) != time.Minute {
		t.Fatalf("expected a 1m step, got %s", end.Sub(start))
	}

	start, end, ok = window(now.Add(2*time.Hour), cursor, true, lag, lookback, maxWindow)
	if !ok || end.Sub(start) != maxWindow {
		t.Fatalf("cap: start %s end %s ok %v", start, end, ok)
	}

	_, _, ok = window(now, now, true, lag, lookback, maxWindow)
	if ok {
		t.Fatal("cursor ahead of the lagged end should be empty")
	}
}

func TestRunAdvancesCursorAfterExport(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	cursor := &memCursor{}
	exported := 0
	fake := &fakeXRay{
		summaries: []string{"1-69fba214-225558552b59105d29f40f79"},
		traces: map[string][]string{
			"1-69fba214-225558552b59105d29f40f79": {
				`{"id":"abcdef1234567890","name":"checkout","start_time":1,"end_time":2,"origin":"AWS::ApiGateway::Stage"}`,
			},
		},
	}
	p := &Poller{
		XRay:     fake,
		Cursor:   cursor,
		Exporter: exporterFunc(func(context.Context, []byte) error { exported++; return nil }),
		Region:   "us-west-2",
		Filter:   DefaultAPIGatewayFilter,
		Logger:   logr.Discard(),
		Sleep:    func(context.Context, time.Duration) error { return nil },
	}
	result, err := p.Run(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Spans != 1 || exported != 1 {
		t.Fatalf("spans %d exports %d", result.Spans, exported)
	}
	if !cursor.stored.Equal(result.End) {
		t.Fatalf("cursor %s end %s", cursor.stored, result.End)
	}
	if fake.filter != DefaultAPIGatewayFilter {
		t.Fatalf("filter %q", fake.filter)
	}
}

func TestRunDoesNotAdvanceCursorWhenExportFails(t *testing.T) {
	cursor := &memCursor{}
	p := &Poller{
		XRay: &fakeXRay{
			summaries: []string{"1-69fba214-225558552b59105d29f40f79"},
			traces: map[string][]string{
				"1-69fba214-225558552b59105d29f40f79": {
					`{"id":"abcdef1234567890","name":"checkout","start_time":1,"end_time":2}`,
				},
			},
		},
		Cursor:   cursor,
		Exporter: exporterFunc(func(context.Context, []byte) error { return errors.New("observe down") }),
		Logger:   logr.Discard(),
		Sleep:    func(context.Context, time.Duration) error { return nil },
	}
	_, err := p.Run(context.Background(), time.Now())
	if err == nil {
		t.Fatal("expected export error")
	}
	if !cursor.stored.IsZero() {
		t.Fatal("cursor advanced after a failed export")
	}
}

func TestBatchGetRetriesUnprocessedAndFollowsNextToken(t *testing.T) {
	fake := &fakeXRay{
		summaries: []string{"1-69fba214-225558552b59105d29f40f79", "1-69fba214-aaaaaaaaaaaaaaaaaaaaaaaa"},
		unprocessedOnce: map[string]bool{
			"1-69fba214-aaaaaaaaaaaaaaaaaaaaaaaa": true,
		},
		traces: map[string][]string{
			"1-69fba214-aaaaaaaaaaaaaaaaaaaaaaaa": {
				`{"id":"bbbbbbbbbbbbbbbb","name":"other","start_time":1,"end_time":2}`,
			},
		},
		page: map[string][]string{
			"1-69fba214-225558552b59105d29f40f79": {
				`{"id":"abcdef1234567890","name":"root","start_time":1,"end_time":2}`,
				`{"id":"1234abcd1234abcd","name":"child","start_time":1,"end_time":2,"parent_id":"abcdef1234567890"}`,
			},
		},
	}
	p := &Poller{
		XRay:     fake,
		Cursor:   &memCursor{},
		Exporter: exporterFunc(func(context.Context, []byte) error { return nil }),
		Logger:   logr.Discard(),
		Sleep:    func(context.Context, time.Duration) error { return nil },
	}
	result, err := p.Run(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.Traces != 2 || result.Spans != 3 {
		t.Fatalf("traces %d spans %d", result.Traces, result.Spans)
	}
	if fake.batchCalls < 3 {
		t.Fatalf("expected a page plus an unprocessed retry, calls %d", fake.batchCalls)
	}
}

type exporterFunc func(context.Context, []byte) error

func (f exporterFunc) Export(ctx context.Context, body []byte) error { return f(ctx, body) }

type memCursor struct {
	stored time.Time
	ok     bool
}

func (m *memCursor) Load(context.Context) (time.Time, bool, error) {
	return m.stored, m.ok, nil
}

func (m *memCursor) Store(_ context.Context, end time.Time) error {
	m.stored = end
	m.ok = true
	return nil
}

type fakeXRay struct {
	summaries       []string
	filter          string
	traces          map[string][]string
	page            map[string][]string
	unprocessedOnce map[string]bool
	batchCalls      int
	summaryPages    int
}

func (f *fakeXRay) GetTraceSummaries(_ context.Context, in *xray.GetTraceSummariesInput, _ ...func(*xray.Options)) (*xray.GetTraceSummariesOutput, error) {
	if in.FilterExpression != nil {
		f.filter = *in.FilterExpression
	}
	f.summaryPages++
	if f.summaryPages == 1 && len(f.summaries) > 1 {
		return &xray.GetTraceSummariesOutput{
			TraceSummaries: []types.TraceSummary{{Id: aws.String(f.summaries[0])}},
			NextToken:      aws.String("page-2"),
		}, nil
	}
	start := 0
	if f.summaryPages > 1 {
		start = 1
	}
	var summaries []types.TraceSummary
	for _, id := range f.summaries[start:] {
		summaries = append(summaries, types.TraceSummary{Id: aws.String(id)})
	}
	return &xray.GetTraceSummariesOutput{TraceSummaries: summaries}, nil
}

func (f *fakeXRay) BatchGetTraces(_ context.Context, in *xray.BatchGetTracesInput, _ ...func(*xray.Options)) (*xray.BatchGetTracesOutput, error) {
	f.batchCalls++
	if in.NextToken == nil {
		var traces []types.Trace
		var unprocessed []string
		for _, id := range in.TraceIds {
			if f.unprocessedOnce[id] {
				delete(f.unprocessedOnce, id)
				unprocessed = append(unprocessed, id)
				continue
			}
			docs := f.docs(id)
			if len(docs) > 1 {
				traces = append(traces, traceWithDocs(id, docs[:1]))
				continue
			}
			traces = append(traces, traceWithDocs(id, docs))
		}
		out := &xray.BatchGetTracesOutput{Traces: traces, UnprocessedTraceIds: unprocessed}
		for _, id := range in.TraceIds {
			if len(f.docs(id)) > 1 && !contains(unprocessed, id) {
				out.NextToken = aws.String("more")
				break
			}
		}
		return out, nil
	}
	var traces []types.Trace
	for _, id := range in.TraceIds {
		docs := f.docs(id)
		if len(docs) > 1 {
			traces = append(traces, traceWithDocs(id, docs[1:]))
		}
	}
	return &xray.BatchGetTracesOutput{Traces: traces}, nil
}

func (f *fakeXRay) docs(id string) []string {
	if docs, ok := f.page[id]; ok {
		return docs
	}
	return f.traces[id]
}

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func traceWithDocs(id string, docs []string) types.Trace {
	segments := make([]types.Segment, len(docs))
	for i, doc := range docs {
		segments[i] = types.Segment{Document: aws.String(doc)}
	}
	return types.Trace{Id: aws.String(id), Segments: segments}
}
