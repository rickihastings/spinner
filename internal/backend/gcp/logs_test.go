package gcp

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

func TestReadLogs_Success(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	logContent := []byte("line 1\nline 2\nline 3\n")
	client.On("ReadObject", ctx, "test-bucket", "my-instance/logs/raw.log").Return(logContent, nil)

	reader, err := readLogs(ctx, client, "test-bucket", "my-instance")
	if err != nil {
		t.Fatalf("readLogs returned error: %v", err)
	}

	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	if string(data) != string(logContent) {
		t.Errorf("expected %q, got %q", string(logContent), string(data))
	}

	client.AssertExpectations(t)
}

func TestReadLogs_ObjectNotFound(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("ReadObject", ctx, "test-bucket", "my-instance/logs/raw.log").
		Return(nil, fmt.Errorf("failed to read object: not found"))

	_, err := readLogs(ctx, client, "test-bucket", "my-instance")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	client.AssertExpectations(t)
}

func TestLogObjectPath(t *testing.T) {
	path := logObjectPath("spinner-myenv-myrepo")
	expected := "spinner-myenv-myrepo/logs/raw.log"

	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestPollLogChunk_NewData(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	bucket := "test-bucket"
	object := "instance/logs/raw.log"

	client.On("ObjectSize", ctx, bucket, object).Return(int64(20), nil)
	client.On("ReadObjectRange", ctx, bucket, object, int64(0)).
		Return([]byte("line 1\nline 2\n"), nil)

	ch := make(chan string, 10)

	newOffset, err := pollLogChunk(ctx, client, bucket, object, 0, ch)
	if err != nil {
		t.Fatalf("pollLogChunk returned error: %v", err)
	}

	if newOffset != 14 {
		t.Errorf("expected offset 14, got %d", newOffset)
	}

	close(ch)

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if lines[0] != "line 1" {
		t.Errorf("expected 'line 1', got %q", lines[0])
	}

	if lines[1] != "line 2" {
		t.Errorf("expected 'line 2', got %q", lines[1])
	}

	client.AssertExpectations(t)
}

func TestPollLogChunk_NoNewData(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	bucket := "test-bucket"
	object := "instance/logs/raw.log"

	client.On("ObjectSize", ctx, bucket, object).Return(int64(10), nil)

	ch := make(chan string, 10)

	newOffset, err := pollLogChunk(ctx, client, bucket, object, 10, ch)
	if err != nil {
		t.Fatalf("pollLogChunk returned error: %v", err)
	}

	if newOffset != 10 {
		t.Errorf("expected offset 10, got %d", newOffset)
	}

	close(ch)

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}

	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}

	client.AssertExpectations(t)
}

func TestPollLogChunk_IncompleteTrailingLine(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	bucket := "test-bucket"
	object := "instance/logs/raw.log"

	// Data ends without a newline — "partial" should not be sent
	client.On("ObjectSize", ctx, bucket, object).Return(int64(25), nil)
	client.On("ReadObjectRange", ctx, bucket, object, int64(0)).
		Return([]byte("complete\npartial"), nil)

	ch := make(chan string, 10)

	newOffset, err := pollLogChunk(ctx, client, bucket, object, 0, ch)
	if err != nil {
		t.Fatalf("pollLogChunk returned error: %v", err)
	}

	// Only "complete\n" should advance the offset (9 bytes)
	if newOffset != 9 {
		t.Errorf("expected offset 9, got %d", newOffset)
	}

	close(ch)

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	if lines[0] != "complete" {
		t.Errorf("expected 'complete', got %q", lines[0])
	}

	client.AssertExpectations(t)
}

func TestWaitForObject_AlreadyExists(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("ObjectExists", ctx, "bucket", "obj").Return(true, nil)

	err := waitForObject(ctx, client, "bucket", "obj")
	if err != nil {
		t.Fatalf("waitForObject returned error: %v", err)
	}

	client.AssertExpectations(t)
}

func TestWaitForObject_ContextCancelled(t *testing.T) {
	client := new(MockGCPClient)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := waitForObject(ctx, client, "bucket", "obj")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPollLogChunk_ResetsOnTruncation(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	bucket := "test-bucket"
	object := "instance/logs/raw.log"

	// Simulate: offset is 500 (from previous iteration) but file was truncated
	// and new data has been written (size is now 20, much less than 500).
	client.On("ObjectSize", ctx, bucket, object).Return(int64(20), nil)
	client.On("ReadObjectRange", ctx, bucket, object, int64(0)).
		Return([]byte("new line 1\nnew line 2\n"), nil)

	ch := make(chan string, 10)

	newOffset, err := pollLogChunk(ctx, client, bucket, object, 500, ch)
	if err != nil {
		t.Fatalf("pollLogChunk returned error: %v", err)
	}

	// Should have reset to 0 and read new content
	if newOffset != 22 {
		t.Errorf("expected offset 22, got %d", newOffset)
	}

	close(ch)

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if lines[0] != "new line 1" {
		t.Errorf("expected 'new line 1', got %q", lines[0])
	}

	if lines[1] != "new line 2" {
		t.Errorf("expected 'new line 2', got %q", lines[1])
	}

	client.AssertExpectations(t)
}

func TestWatchLogs_ContextCancellation(t *testing.T) {
	client := new(MockGCPClient)
	ctx, cancel := context.WithCancel(context.Background())

	// Object exists immediately
	client.On("ObjectExists", mock.Anything, "bucket", "inst/logs/raw.log").Return(true, nil)
	// Return no data so the loop just polls
	client.On("ObjectSize", mock.Anything, "bucket", "inst/logs/raw.log").Return(int64(0), nil)

	ch := make(chan string, 10)

	done := make(chan error, 1)

	go func() {
		done <- watchLogs(ctx, client, "bucket", "inst", ch)
	}()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
