package metricsconfigurator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

const testHandleDeleteStackId = "arn:aws:cloudformation:us-west-2:000000000000:stack/MyStack/abc123"

// stream is a small constructor for awstest.MetricStreamRecord that
// keeps the test bodies readable.
func stream(name string, stackIdTagValue string) awstest.MetricStreamRecord {
	rec := awstest.MetricStreamRecord{
		Name: name,
		Arn:  "arn:aws:cloudwatch:us-west-2:000000000000:metric-stream/" + name,
	}
	if stackIdTagValue != "" {
		rec.Tags = []types.Tag{{Key: aws.String(stackIdTagKey), Value: aws.String(stackIdTagValue)}}
	}
	return rec
}

// streamWithTags lets a test pin arbitrary tags on a stream record
// (e.g. tags other than ours).
func streamWithTags(name string, tags []types.Tag) awstest.MetricStreamRecord {
	return awstest.MetricStreamRecord{
		Name: name,
		Arn:  "arn:aws:cloudwatch:us-west-2:000000000000:metric-stream/" + name,
		Tags: tags,
	}
}

// newDeleteHandler returns a Handler wired to the given fake CloudWatch
// client. The CFN response URL points at a discarding test server so
// reportStatus calls succeed.
func newDeleteHandler(t *testing.T, cw *awstest.CloudWatchClient) (Handler, *httptest.Server) {
	t.Helper()
	cfn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(cfn.Close)
	h := Handler{
		Logger:              logr.Discard(),
		NewCloudWatchClient: func(aws.Config) cwAPI { return cw },
	}
	return h, cfn
}

func deleteRequest(cfnURL string) *Request {
	return &Request{
		RequestType:       "Delete",
		ResponseURL:       cfnURL,
		StackId:           testHandleDeleteStackId,
		RequestId:         "req-1",
		LogicalResourceId: "MetricStream",
	}
}

// streamNames returns the names of streams currently in the fake, in order.
func streamNames(cw *awstest.CloudWatchClient) []string {
	names := make([]string, len(cw.Streams))
	for i, s := range cw.Streams {
		names[i] = s.Name
	}
	return names
}

func TestHandleDelete_OnlyDeletesOwnStreams(t *testing.T) {
	// Mix four streams in the account:
	//   - one tagged with our stack id (must be deleted)
	//   - one tagged with a different stack id (must be left alone)
	//   - one tagged but with a non-stack-id key (must be left alone)
	//   - one with no tags at all (must be left alone)
	cw := &awstest.CloudWatchClient{
		Streams: []awstest.MetricStreamRecord{
			stream("ours", testHandleDeleteStackId),
			stream("other-stack", "arn:aws:cloudformation:us-west-2:000000000000:stack/OtherStack/zzz"),
			streamWithTags("customer-managed", []types.Tag{
				{Key: aws.String("Owner"), Value: aws.String("customer")},
			}),
			streamWithTags("untagged", nil),
		},
	}
	h, cfn := newDeleteHandler(t, cw)

	if _, err := h.handleDelete(context.Background(), aws.Config{}, deleteRequest(cfn.URL)); err != nil {
		t.Fatalf("handleDelete returned error: %v", err)
	}

	got := streamNames(cw)
	want := []string{"other-stack", "customer-managed", "untagged"}
	if !equalSlices(got, want) {
		t.Errorf("surviving streams = %v, want %v", got, want)
	}
}

func TestHandleDelete_Pagination(t *testing.T) {
	// Five streams all owned by us, returned across multiple pages.
	streams := make([]awstest.MetricStreamRecord, 5)
	for i := range streams {
		streams[i] = stream(streamNameFromIdx(i), testHandleDeleteStackId)
	}
	cw := &awstest.CloudWatchClient{Streams: streams, PageSize: 2}
	h, cfn := newDeleteHandler(t, cw)

	if _, err := h.handleDelete(context.Background(), aws.Config{}, deleteRequest(cfn.URL)); err != nil {
		t.Fatalf("handleDelete returned error: %v", err)
	}

	if got := streamNames(cw); len(got) != 0 {
		t.Errorf("expected all streams deleted, still have: %v", got)
	}
}

func TestHandleDelete_ListTagsErrorIsTolerated(t *testing.T) {
	// Three streams, all owned by us. ListTagsForResource fails on the
	// middle stream; the surrounding streams must still be deleted, the
	// failing one must be left alone (we can't prove ownership).
	cw := &awstest.CloudWatchClient{
		Streams: []awstest.MetricStreamRecord{
			stream("first", testHandleDeleteStackId),
			stream("middle", testHandleDeleteStackId),
			stream("last", testHandleDeleteStackId),
		},
	}
	cw.ListTagsForResourceFunc = func(_ context.Context, in *cloudwatch.ListTagsForResourceInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error) {
		if aws.ToString(in.ResourceARN) == "arn:aws:cloudwatch:us-west-2:000000000000:metric-stream/middle" {
			return nil, errors.New("transient list-tags failure")
		}
		// Fall back to seeded tags for the other streams.
		for _, s := range cw.Streams {
			if s.Arn == aws.ToString(in.ResourceARN) {
				return &cloudwatch.ListTagsForResourceOutput{Tags: append([]types.Tag(nil), s.Tags...)}, nil
			}
		}
		return &cloudwatch.ListTagsForResourceOutput{}, nil
	}
	h, cfn := newDeleteHandler(t, cw)

	if _, err := h.handleDelete(context.Background(), aws.Config{}, deleteRequest(cfn.URL)); err != nil {
		t.Fatalf("handleDelete returned error: %v", err)
	}

	got := streamNames(cw)
	want := []string{"middle"}
	if !equalSlices(got, want) {
		t.Errorf("surviving streams = %v, want %v (only the list-tags-failed stream should remain)", got, want)
	}
}

func streamNameFromIdx(i int) string {
	return string(rune('a' + i))
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
