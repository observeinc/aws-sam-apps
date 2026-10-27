package xraypoller

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	batchSize              = 5
	maxUnprocessedAttempts = 3
	defaultSafetyLag       = 90 * time.Second
	defaultInitialLookback = 60 * time.Second
	defaultMaxWindow       = 15 * time.Minute
)

// xrayAPI is the subset of the X-Ray client this poller uses.
type xrayAPI interface {
	GetTraceSummaries(context.Context, *xray.GetTraceSummariesInput, ...func(*xray.Options)) (*xray.GetTraceSummariesOutput, error)
	BatchGetTraces(context.Context, *xray.BatchGetTracesInput, ...func(*xray.Options)) (*xray.BatchGetTracesOutput, error)
}

// CursorStore remembers the end of the last successful window so the next
// poll does not re-read it.
type CursorStore interface {
	Load(ctx context.Context) (time.Time, bool, error)
	Store(ctx context.Context, end time.Time) error
}

// Exporter delivers converted traces. Implementations batch the upload so
// one polling window cannot become a single unbounded request.
type Exporter interface {
	Export(ctx context.Context, traces ptrace.Traces) error
}

// Poller reads a window of X-Ray traces and exports them as OTLP.
type Poller struct {
	XRay            xrayAPI
	Cursor          CursorStore
	Exporter        Exporter
	Region          string
	ScopeVersion    string
	Filter          string
	SafetyLag       time.Duration
	InitialLookback time.Duration
	MaxWindow       time.Duration
	Now             func() time.Time
	Sleep           func(context.Context, time.Duration) error
	Logger          logr.Logger
}

// ExcludeService narrows filter so traces that touch service are skipped.
// The poller traces itself, and an empty filter would otherwise forward one
// poller trace per invocation.
func ExcludeService(filter, service string) string {
	if service == "" {
		return filter
	}
	exclude := fmt.Sprintf("!service(%q)", service)
	if filter == "" {
		return exclude
	}
	return fmt.Sprintf("(%s) AND %s", filter, exclude)
}

// Result counts what one invocation forwarded.
type Result struct {
	Traces int
	Spans  int
	Start  time.Time
	End    time.Time
}

// Run polls (cursor, now-safetyLag], capped to MaxWindow, and advances the
// cursor only after a successful export.
func (p *Poller) Run(ctx context.Context, now time.Time) (Result, error) {
	if now.IsZero() {
		now = p.now()
	}
	cursor, ok, err := p.Cursor.Load(ctx)
	if err != nil {
		return Result{}, err
	}
	start, end, nonempty := window(now, cursor, ok, p.safetyLag(), p.initialLookback(), p.maxWindow())
	result := Result{Start: start, End: end}
	if !nonempty {
		p.Logger.V(1).Info("xray window empty", "start", start, "end", end)
		return result, nil
	}

	ids, err := p.listTraceIDs(ctx, start, end)
	if err != nil {
		return result, err
	}
	result.Traces = len(ids)
	if len(ids) == 0 {
		if err := p.Cursor.Store(ctx, end); err != nil {
			return result, err
		}
		p.Logger.Info("no x-ray traces in window", "start", start, "end", end)
		return result, nil
	}

	traces, err := p.batchGet(ctx, ids)
	if err != nil {
		return result, err
	}
	batch := NewTraceBatch()
	for _, trace := range traces {
		if trace.Id == nil {
			continue
		}
		docs := make([]string, 0, len(trace.Segments))
		for _, segment := range trace.Segments {
			if segment.Document != nil {
				docs = append(docs, *segment.Document)
			}
		}
		n, err := batch.Add(*trace.Id, docs, p.Region, p.scopeVersion())
		if err != nil {
			// A single bad document must not pin the cursor. The next poll
			// would read the same window and fail the same way.
			p.Logger.Error(err, "skipping x-ray trace", "trace", *trace.Id)
			continue
		}
		result.Spans += n
	}

	if result.Spans > 0 {
		if err := p.Exporter.Export(ctx, batch.Traces()); err != nil {
			return result, err
		}
	}
	if err := p.Cursor.Store(ctx, end); err != nil {
		return result, err
	}
	p.Logger.Info("forwarded x-ray spans", "traces", result.Traces, "spans", result.Spans, "start", start, "end", end)
	return result, nil
}

func (p *Poller) listTraceIDs(ctx context.Context, start, end time.Time) ([]string, error) {
	var ids []string
	var token *string
	for {
		in := &xray.GetTraceSummariesInput{
			StartTime: aws.Time(start),
			EndTime:   aws.Time(end),
			Sampling:  aws.Bool(false),
			// Event time, not the trace-id timestamp. A segment that arrives
			// after the trace started is selected again when its event time
			// falls in a later window. Span ids are stable, so reposting the
			// earlier spans does not create a second copy.
			TimeRangeType: types.TimeRangeTypeEvent,
			NextToken:     token,
		}
		if p.Filter != "" {
			in.FilterExpression = aws.String(p.Filter)
		}
		out, err := p.XRay.GetTraceSummaries(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("get trace summaries: %w", err)
		}
		for _, summary := range out.TraceSummaries {
			if summary.Id != nil && *summary.Id != "" {
				ids = append(ids, *summary.Id)
			}
		}
		if out.NextToken == nil || *out.NextToken == "" {
			return ids, nil
		}
		token = out.NextToken
	}
}

func (p *Poller) batchGet(ctx context.Context, ids []string) ([]types.Trace, error) {
	pending := ids
	var traces []types.Trace
	for attempt := 1; len(pending) > 0; attempt++ {
		if attempt > maxUnprocessedAttempts {
			return nil, fmt.Errorf("x-ray left %d traces unprocessed", len(pending))
		}
		if attempt > 1 {
			if err := p.sleep(ctx, time.Duration(attempt-1)*time.Second); err != nil {
				return nil, err
			}
		}
		var still []string
		for _, chunk := range chunks(pending, batchSize) {
			var page *string
			first := true
			for {
				out, err := p.XRay.BatchGetTraces(ctx, &xray.BatchGetTracesInput{
					TraceIds:  chunk,
					NextToken: page,
				})
				if err != nil {
					return nil, fmt.Errorf("batch get traces: %w", err)
				}
				traces = append(traces, out.Traces...)
				if first {
					still = append(still, out.UnprocessedTraceIds...)
					first = false
				}
				if out.NextToken == nil || *out.NextToken == "" {
					break
				}
				page = out.NextToken
			}
		}
		pending = still
	}
	return traces, nil
}

// window returns the half-open range of event times to read. The end is
// lagged so X-Ray has time to index segments. A stored cursor is the previous
// end, so successive polls do not overlap. A large gap is capped so one
// invocation cannot try to drain days of traces.
func window(now, cursor time.Time, hasCursor bool, safetyLag, initialLookback, maxWindow time.Duration) (start, end time.Time, ok bool) {
	end = now.Add(-safetyLag)
	if hasCursor {
		start = cursor
	} else {
		start = end.Add(-initialLookback)
	}
	if !start.Before(end) {
		return start, end, false
	}
	if maxWindow > 0 && end.Sub(start) > maxWindow {
		end = start.Add(maxWindow)
	}
	return start, end, true
}

func chunks(ids []string, size int) [][]string {
	var out [][]string
	for len(ids) > 0 {
		n := size
		if n > len(ids) {
			n = len(ids)
		}
		out = append(out, ids[:n])
		ids = ids[n:]
	}
	return out
}

func (p *Poller) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

func (p *Poller) sleep(ctx context.Context, d time.Duration) error {
	if p.Sleep != nil {
		return p.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (p *Poller) safetyLag() time.Duration {
	if p.SafetyLag > 0 {
		return p.SafetyLag
	}
	return defaultSafetyLag
}

func (p *Poller) initialLookback() time.Duration {
	if p.InitialLookback > 0 {
		return p.InitialLookback
	}
	return defaultInitialLookback
}

func (p *Poller) maxWindow() time.Duration {
	if p.MaxWindow > 0 {
		return p.MaxWindow
	}
	return defaultMaxWindow
}

func (p *Poller) scopeVersion() string {
	if p.ScopeVersion != "" {
		return p.ScopeVersion
	}
	return "0.0.0"
}
