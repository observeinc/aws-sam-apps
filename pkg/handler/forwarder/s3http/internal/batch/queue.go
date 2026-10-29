package batch

import (
	"bytes"
	"compress/gzip"
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
	MaxBatchSize int  // maximum batch size in bytes
	Capacity     int  // channel capacity
	GzipLevel    *int // gzip compression level
	Prefix       []byte
	Delimiter    []byte
	Suffix       []byte
}

// Queue appends item to buffer until batch size is reached.
type Queue struct {

	// the chunk currently being appended to
	buffer *bytes.Buffer

	// chunk may be compressed, so we write and flush via the following
	// accessors
	writer io.Writer
	closer io.Closer

	newWriterFunc func(*bytes.Buffer) (io.Writer, io.Closer)
	written       int
	maxBatchSize  int
	ch            chan *bytes.Buffer // channel containing batches
	prefix        []byte
	delimiter     []byte
	suffix        []byte
	records       int
}

// Push a record to queue for batching.
func (q *Queue) Push(ctx context.Context, record []byte) error {
	additional := len(record) + len(q.suffix)
	if q.records == 0 {
		additional += len(q.prefix)
	} else {
		additional += len(q.delimiter)
	}
	if q.maxBatchSize > 0 && q.written+additional > q.maxBatchSize {
		if q.records == 0 {
			return fmt.Errorf("%w: %d", ErrRecordLenExceedsBatchSize, len(record))
		}

		if err := q.flush(ctx); err != nil {
			return fmt.Errorf("failed to flush batch: %w", err)
		}
		if len(q.prefix)+len(record)+len(q.suffix) > q.maxBatchSize {
			return fmt.Errorf("%w: %d", ErrRecordLenExceedsBatchSize, len(record))
		}
	}

	if q.records == 0 {
		buf, ok := bufPool.Get().(*bytes.Buffer)
		if !ok {
			panic("failed type assertion")
		}
		buf.Reset()
		q.buffer = buf
		q.writer, q.closer = q.newWriterFunc(buf)
		if len(q.prefix) > 0 {
			n, err := q.writer.Write(q.prefix)
			if err != nil {
				return fmt.Errorf("failed to buffer prefix: %w", err)
			}
			q.written += n
		}
	} else if len(q.delimiter) > 0 {
		n, err := q.writer.Write(q.delimiter)
		if err != nil {
			return fmt.Errorf("failed to buffer delimiter: %w", err)
		}
		q.written += n
	}

	n, err := q.writer.Write(record)
	if err != nil {
		return fmt.Errorf("failed to buffer record: %w", err)
	}
	q.written += n
	q.records++
	return nil
}

func (q *Queue) flush(ctx context.Context) error {
	if q.records == 0 {
		return nil
	}
	if len(q.suffix) > 0 {
		if _, err := q.writer.Write(q.suffix); err != nil {
			return fmt.Errorf("failed to buffer suffix: %w", err)
		}
	}

	if err := q.closer.Close(); err != nil {
		return fmt.Errorf("failed to close buffer: %w", err)
	}

	q.written = 0
	q.records = 0

	select {
	case q.ch <- q.buffer:
		return nil
	case <-ctx.Done():
		bufPool.Put(q.buffer)
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

	q := &Queue{
		newWriterFunc: func(buf *bytes.Buffer) (io.Writer, io.Closer) {
			return buf, io.NopCloser(buf)
		},
		maxBatchSize: cfg.MaxBatchSize,
		prefix:       cfg.Prefix,
		delimiter:    cfg.Delimiter,
		suffix:       cfg.Suffix,
		ch:           make(chan *bytes.Buffer, cfg.Capacity),
	}

	if cfg.GzipLevel != nil {
		q.newWriterFunc = func(buf *bytes.Buffer) (io.Writer, io.Closer) {
			gw, _ := gzip.NewWriterLevel(buf, *cfg.GzipLevel)
			return gw, gw
		}
	}

	return q

}
