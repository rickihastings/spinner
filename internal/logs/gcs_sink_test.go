package logs

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockObjectWriter records all WriteObject calls for assertions.
type mockObjectWriter struct {
	mu    sync.Mutex
	calls []writeCall
	err   error
}

type writeCall struct {
	bucket string
	object string
	data   []byte
}

func (m *mockObjectWriter) WriteObject(_ context.Context, bucket, object string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := make([]byte, len(data))
	copy(copied, data)

	m.calls = append(m.calls, writeCall{bucket: bucket, object: object, data: copied})

	return m.err
}

func (m *mockObjectWriter) lastData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.calls) == 0 {
		return nil
	}

	return m.calls[len(m.calls)-1].data
}

func (m *mockObjectWriter) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.calls)
}

func TestGCSSink_Write(t *testing.T) {
	writer := &mockObjectWriter{}
	ctx := context.Background()

	sink := NewGCSSink(ctx, writer, "test-bucket", "test/log.txt")

	n, err := sink.Write([]byte("hello "))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if n != 6 {
		t.Errorf("expected 6 bytes written, got %d", n)
	}

	n, err = sink.Write([]byte("world"))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	// Close triggers final flush
	if err := sink.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	data := writer.lastData()
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(data))
	}
}

func TestGCSSink_Close_FlushesToCorrectBucketAndObject(t *testing.T) {
	writer := &mockObjectWriter{}
	ctx := context.Background()

	sink := NewGCSSink(ctx, writer, "my-bucket", "instance/logs/raw.log")

	_, _ = sink.Write([]byte("data"))

	if err := sink.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	writer.mu.Lock()
	defer writer.mu.Unlock()

	if len(writer.calls) == 0 {
		t.Fatal("expected at least one WriteObject call")
	}

	last := writer.calls[len(writer.calls)-1]
	if last.bucket != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got %q", last.bucket)
	}

	if last.object != "instance/logs/raw.log" {
		t.Errorf("expected object 'instance/logs/raw.log', got %q", last.object)
	}
}

func TestGCSSink_Close_NoWriteWhenEmpty(t *testing.T) {
	writer := &mockObjectWriter{}
	ctx := context.Background()

	sink := NewGCSSink(ctx, writer, "test-bucket", "test/log.txt")

	if err := sink.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	if writer.callCount() != 0 {
		t.Errorf("expected no WriteObject calls for empty sink, got %d", writer.callCount())
	}
}

func TestGCSSink_PeriodicFlush(t *testing.T) {
	writer := &mockObjectWriter{}
	ctx := context.Background()

	sink := NewGCSSink(ctx, writer, "test-bucket", "test/log.txt")
	// Use a shorter interval for testing
	sink.cancel()
	sink.wg.Wait()

	// Restart with short interval
	sink.interval = 50 * time.Millisecond
	sinkCtx, cancel := context.WithCancel(ctx)
	sink.ctx = sinkCtx
	sink.cancel = cancel
	sink.wg.Add(1)

	go sink.flushLoop()

	_, _ = sink.Write([]byte("first"))

	// Wait for at least one periodic flush
	time.Sleep(150 * time.Millisecond)

	if writer.callCount() < 1 {
		t.Error("expected at least one periodic flush")
	}

	// Write more data
	_, _ = sink.Write([]byte(" second"))

	// Wait for another flush
	time.Sleep(150 * time.Millisecond)

	if err := sink.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	// The last flush should have all accumulated data
	data := writer.lastData()
	if string(data) != "first second" {
		t.Errorf("expected 'first second', got %q", string(data))
	}
}

func TestGCSSink_ConcurrentWrites(t *testing.T) {
	writer := &mockObjectWriter{}
	ctx := context.Background()

	sink := NewGCSSink(ctx, writer, "test-bucket", "test/log.txt")

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, _ = sink.Write([]byte("x"))
		}()
	}

	wg.Wait()

	if err := sink.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	data := writer.lastData()
	if len(data) != 100 {
		t.Errorf("expected 100 bytes, got %d", len(data))
	}
}

func TestGCSSink_FlushSkipsWhenNotDirty(t *testing.T) {
	writer := &mockObjectWriter{}
	ctx := context.Background()

	sink := NewGCSSink(ctx, writer, "test-bucket", "test/log.txt")

	_, _ = sink.Write([]byte("data"))

	// First flush should write
	if err := sink.flush(); err != nil {
		t.Fatalf("flush returned error: %v", err)
	}

	count := writer.callCount()
	if count != 1 {
		t.Fatalf("expected 1 call after first flush, got %d", count)
	}

	// Second flush without new writes should be a no-op
	if err := sink.flush(); err != nil {
		t.Fatalf("flush returned error: %v", err)
	}

	if writer.callCount() != 1 {
		t.Errorf("expected no additional calls, got %d total", writer.callCount())
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
