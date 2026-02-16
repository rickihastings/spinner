// Package logs provides log streaming utilities for Spinner backends.
package logs

import (
	"context"
	"sync"
	"time"
)

const (
	// defaultFlushInterval is the default interval between GCS flushes.
	defaultFlushInterval = 2 * time.Second
)

// ObjectWriter is the minimal interface needed for writing to an object store.
// This enables dependency injection and testing without real GCS.
type ObjectWriter interface {
	WriteObject(ctx context.Context, bucket, object string, data []byte) error
}

// GCSSink implements io.Writer by buffering writes and periodically
// flushing accumulated content to a GCS object. Each flush overwrites
// the full object with all content received so far.
type GCSSink struct {
	writer   ObjectWriter
	bucket   string
	object   string
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	buf      []byte
	dirty    bool
	wg       sync.WaitGroup
	interval time.Duration
}

// NewGCSSink creates a new GCSSink that flushes to the given GCS bucket/object.
// The sink starts a background goroutine that flushes every 2 seconds.
// Call Close() to do a final flush and stop the background goroutine.
func NewGCSSink(ctx context.Context, writer ObjectWriter, bucket, object string) *GCSSink {
	sinkCtx, cancel := context.WithCancel(ctx)

	s := &GCSSink{
		writer:   writer,
		bucket:   bucket,
		object:   object,
		ctx:      sinkCtx,
		cancel:   cancel,
		buf:      make([]byte, 0, 4096),
		interval: defaultFlushInterval,
	}

	s.wg.Add(1)

	go s.flushLoop()

	return s
}

// Write appends data to the internal buffer. It never blocks on GCS I/O.
// Write is safe for concurrent use.
func (s *GCSSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf = append(s.buf, p...)
	s.dirty = true

	return len(p), nil
}

// Close performs a final flush and stops the background goroutine.
func (s *GCSSink) Close() error {
	s.cancel()
	s.wg.Wait()

	return s.flush()
}

// flush writes the buffered content to GCS if there is new data.
func (s *GCSSink) flush() error {
	s.mu.Lock()

	if !s.dirty {
		s.mu.Unlock()
		return nil
	}

	// Copy the buffer so we can release the lock before the network call.
	data := make([]byte, len(s.buf))
	copy(data, s.buf)
	s.dirty = false
	s.mu.Unlock()

	// Use a background context for the actual write since the sink context
	// may be cancelled during Close(), and we still want the final flush
	// to succeed.
	return s.writer.WriteObject(context.Background(), s.bucket, s.object, data)
}

// flushLoop runs periodic flushes until the context is cancelled.
func (s *GCSSink) flushLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			_ = s.flush()
		}
	}
}
