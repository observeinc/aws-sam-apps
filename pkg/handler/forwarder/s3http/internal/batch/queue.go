package batch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	ErrRecordLenExceedsBatchSize = errors.New("record larger than max batch size")

	bufPool = sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}
)

type QueueConfig struct {
	MaxBatchSize int // maximum batch size in bytes
	Capacity     int // channel capacity
	Delimiter    []byte
}

// Queue appends item to buffer until batch size is reached.
type Queue struct {
	buffer       bytes.Buffer
	maxBatchSize int
	ch           chan *bytes.Buffer // channel containing batches
	delimiter    []byte
}

// Push a record to queue for batching.
// We assume record includes delimiter.
func (q *Queue) Push(ctx context.Context, record []byte) error {
	if q.maxBatchSize > 0 && q.buffer.Len()+len(record)+len(q.delimiter) > q.maxBatchSize {
		if len(record) > q.maxBatchSize {
			return fmt.Errorf("%w: %d", ErrRecordLenExceedsBatchSize, len(record))
		}

		if err := q.flush(ctx); err != nil {
			return fmt.Errorf("failed to flush batch: %w", err)
		}
	}

	if _, err := q.buffer.Write(record); err != nil {
		return fmt.Errorf("failed to buffer record: %w", err)
	}

	if len(q.delimiter) > 0 {
		if _, err := q.buffer.Write(q.delimiter); err != nil {
			return fmt.Errorf("failed to buffer delimiter: %w", err)
		}
	}
	return nil
}

func (q *Queue) flush(ctx context.Context) error {
	if q.buffer.Len() == 0 {
		return nil
	}

	buf, ok := bufPool.Get().(*bytes.Buffer)
	if !ok {
		panic("failed type assertion")
	}

	if _, err := io.Copy(buf, &q.buffer); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}

	select {
	case q.ch <- buf:
		return nil
	case <-ctx.Done():
		bufPool.Put(buf)
		return fmt.Errorf("cancelled flush: %w", ctx.Err())
	}
}

// Process items from batch queue, until queue is either closed or context is cancelled.
func (q *Queue) Process(ctx context.Context, c Handler) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cancelled process: %w", ctx.Err())
		case buf, ok := <-q.ch:
			if !ok {
				return nil
			}
			err := c.Handle(ctx, buf)
			bufPool.Put(buf)
			if err != nil {
				// nolint:wrapcheck
				return err
			}
		}
	}
}

// Close the queue.
// Flushing batches after close will currently result in panic.
func (q *Queue) Close() error {
	if err := q.flush(context.Background()); err != nil {
		return fmt.Errorf("failed to close: %w", err)
	}
	close(q.ch)
	return nil
}

func NewQueue(cfg *QueueConfig) *Queue {
	if cfg == nil {
		cfg = &QueueConfig{}
	}

	return &Queue{
		maxBatchSize: cfg.MaxBatchSize,
		delimiter:    cfg.Delimiter,
		ch:           make(chan *bytes.Buffer, cfg.Capacity),
	}
}
