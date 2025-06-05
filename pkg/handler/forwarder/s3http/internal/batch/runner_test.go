package batch_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http/internal/batch"
)

func ptr[T any](v T) *T {
	return &v
}

func TestRunner(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		*batch.RunInput
		Input           string
		ExpectedBatches []string
		ExpectedError   error
	}{
		{
			RunInput: &batch.RunInput{
				MaxBatchSize: ptr(38),
			},
			Input: `
			{"hello": "world"}
			{"hello": "world"}
			{"hello": "world"}
			{"hello": "world"}
			{"hello": "world"}
			{"hello": "world"}
			`,
			ExpectedBatches: []string{
				"{\"hello\": \"world\"}\n{\"hello\": \"world\"}\n",
				"{\"hello\": \"world\"}\n{\"hello\": \"world\"}\n",
				"{\"hello\": \"world\"}\n{\"hello\": \"world\"}\n",
			},
		},
	}

	for i, tc := range testcases {
		tt := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			var batches []string
			var mu sync.Mutex

			tt.Decoder = json.NewDecoder(strings.NewReader(tt.Input))
			tt.Handler = batch.HandlerFunc(func(_ context.Context, r io.Reader) error {
				data, err := io.ReadAll(r)
				if err != nil {
					return fmt.Errorf("failed to read: %w", err)
				}
				mu.Lock()
				batches = append(batches, string(data))
				mu.Unlock()
				return nil
			})

			if err := batch.Run(context.Background(), tt.RunInput); err != nil {
				if diff := cmp.Diff(err, tt.ExpectedError, cmpopts.EquateErrors()); diff != "" {
					t.Error("unexpected error", diff)
				}
			} else if diff := cmp.Diff(batches, tt.ExpectedBatches); diff != "" {
				t.Error("unexpected result", diff)
			}
		})
	}
}
