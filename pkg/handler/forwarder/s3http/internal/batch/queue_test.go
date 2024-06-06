package batch_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/sync/errgroup"

	"github.com/observeinc/aws-sam-apps/pkg/handler/forwarder/s3http/internal/batch"
)

func TestQueue(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		*batch.QueueConfig
		Records         []string
		ExpectedBatches []string
		ExpectedError   error
	}{
		{
			QueueConfig: &batch.QueueConfig{
				MaxBatchSize: 12,
				Capacity:     1,
			},
			Records: []string{
				"hello\n",
				"world\n",
				"ok\n",
			},
			ExpectedBatches: []string{
				"hello\nworld\n",
				"ok\n",
			},
		},
		{
			QueueConfig: &batch.QueueConfig{
				MaxBatchSize: 5,
				Capacity:     1,
			},
			Records: []string{
				"too long\n",
			},
			ExpectedError: batch.ErrRecordLenExceedsBatchSize,
		},
		{
			QueueConfig: &batch.QueueConfig{
				MaxBatchSize: 15,
				Capacity:     1,
			},
			Records: []string{
				"hello world\n",
				"ok\n",
				"hello world\n",
			},
			ExpectedBatches: []string{
				"hello world\nok\n",
				"hello world\n",
			},
		},
	}

	for i, tc := range testcases {
		tt := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			q := batch.NewQueue(tt.QueueConfig)
			g, ctx := errgroup.WithContext(context.Background())

			var batches []string

			// writer
			g.Go(func() error {
				err := q.Process(ctx, batch.HandlerFunc(func(_ context.Context, r io.Reader) error {
					data, err := io.ReadAll(r)
					if err != nil {
						return fmt.Errorf("failed to read: %w", err)
					}
					batches = append(batches, string(data))
					return nil
				}))
				if err != nil {
					return fmt.Errorf("failed to consume: %w", err)
				}
				return nil
			})

			// reader
			g.Go(func() error {
				for _, record := range tt.Records {
					if err := q.Push(ctx, []byte(record)); err != nil {
						return fmt.Errorf("failed to push record: %w", err)
					}
				}
				if err := q.Close(); err != nil {
					return fmt.Errorf("failed to close: %w", err)
				}
				return nil
			})

			if err := g.Wait(); err != nil {
				if diff := cmp.Diff(err, tt.ExpectedError, cmpopts.EquateErrors()); diff != "" {
					t.Error("unexpected error", diff)
				}
			} else if diff := cmp.Diff(batches, tt.ExpectedBatches); diff != "" {
				t.Error("unexpected result", diff)
			}
		})
	}
}

func BenchmarkQueue(b *testing.B) {
	capacity := 100
	q := batch.NewQueue(&batch.QueueConfig{
		MaxBatchSize: 12,
		Capacity:     capacity,
	})

	g, ctx := errgroup.WithContext(context.Background())

	for range capacity {
		g.Go(func() error {
			err := q.Process(ctx, batch.HandlerFunc(func(_ context.Context, r io.Reader) error {
				if _, err := io.ReadAll(r); err != nil {
					return fmt.Errorf("failed to read: %w", err)
				}
				return nil
			}))
			if err != nil {
				return fmt.Errorf("consume returned an error: %w", err)
			}
			return nil
		})
	}

	g.Go(func() error {
		for range b.N {
			if err := q.Push(ctx, []byte("hello world\n")); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}
		if err := q.Close(); err != nil {
			return fmt.Errorf("failed to close: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		b.Fatal(err)
	}
}
