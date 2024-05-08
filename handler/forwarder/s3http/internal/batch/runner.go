package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/sync/errgroup"
)

const (
	defaultMaxRecordSize  int = 4e6
	defaultMaxBatchSize   int = 1e7
	defaultMaxConcurrency     = 4
	defaultCapacityFactor     = 2
)

type Decoder interface {
	More() bool
	Decode(any) error
}

type Handler interface {
	Handle(context.Context, io.Reader) error
}

type HandlerFunc func(context.Context, io.Reader) error

func (fn HandlerFunc) Handle(ctx context.Context, r io.Reader) error {
	return fn(ctx, r)
}

var _ Handler = HandlerFunc(nil)

type RunInput struct {
	Decoder
	Handler

	MaxConcurrency *int // how many handlers to run concurrenctly
	MaxBatchSize   *int // maximum size in bytes for each batch
	MaxRecordSize  *int // maximum size in bytes for each record
	CapacityFactor *int // channel capacity, calculated as a multiple of concurrency
}

// Run processes all events from a decoder and feeds them into 1 or more batch handlers.
func Run(ctx context.Context, r *RunInput) error {
	if r == nil {
		return nil
	}

	g, ctx := errgroup.WithContext(ctx)

	var (
		maxConcurrency = defaultMaxConcurrency
		maxBatchSize   = defaultMaxBatchSize
		maxRecordSize  = defaultMaxRecordSize
		capacityFactor = defaultCapacityFactor
	)

	if v := r.MaxConcurrency; v != nil && *v > 0 {
		maxConcurrency = *v
	}
	if v := r.MaxBatchSize; v != nil && *v > 0 {
		maxBatchSize = *v
	}
	if v := r.MaxRecordSize; v != nil && *v > 0 {
		maxRecordSize = *v
	}
	if v := r.CapacityFactor; v != nil && *v > 0 {
		capacityFactor = *v
	}

	q := NewQueue(&QueueConfig{
		MaxBatchSize: maxBatchSize,
		Capacity:     capacityFactor * maxConcurrency,
		Delimiter:    []byte("\n"),
	})

	for range maxConcurrency {
		g.Go(func() error { return q.Process(ctx, r.Handler) })
	}

	g.Go(func() error {
		var v json.RawMessage
		for r.More() {
			if err := r.Decode(&v); err != nil {
				return fmt.Errorf("failed to decode: %w", err)
			}

			if maxRecordSize > 0 && len(v) > maxRecordSize {
				continue
			}

			if err := q.Push(ctx, v); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}
		if err := q.Close(); err != nil {
			return fmt.Errorf("failed to close queue: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to stream data: %w", err)
	}
	return nil
}
