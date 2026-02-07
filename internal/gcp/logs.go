package gcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"
)

const (
	// logPollInterval is how often WatchLogs polls GCS for new content.
	logPollInterval = 2 * time.Second

	// logObjectWaitTimeout is the max time WatchLogs waits for the log object
	// to appear before returning an error.
	logObjectWaitTimeout = 5 * time.Minute
)

// logObjectPath returns the GCS object path for an instance's raw log.
func logObjectPath(instanceName string) string {
	return instanceName + "/logs/raw.log"
}

// readLogs downloads the full log object from GCS and returns it as an io.ReadCloser.
func readLogs(ctx context.Context, client Client, bucket, instanceName string) (io.ReadCloser, error) {
	object := logObjectPath(instanceName)

	data, err := client.ReadObject(ctx, bucket, object)
	if err != nil {
		return nil, fmt.Errorf("failed to read log object gs://%s/%s: %w", bucket, object, err)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// watchLogs polls GCS for new log content and sends lines to the channel.
// It tracks a byte offset and uses range reads to fetch only new data.
// The channel is NOT closed by this function; the caller owns the channel lifecycle.
func watchLogs(ctx context.Context, client Client, bucket, instanceName string, ch chan<- string) error {
	object := logObjectPath(instanceName)

	// Wait for the log object to appear.
	if err := waitForObject(ctx, client, bucket, object); err != nil {
		return err
	}

	var offset int64

	ticker := time.NewTicker(logPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			newOffset, err := pollLogChunk(ctx, client, bucket, object, offset, ch)
			if err != nil {
				// Log read errors are transient (e.g., GCS hiccup); keep polling.
				continue
			}

			offset = newOffset
		}
	}
}

// waitForObject polls until the GCS object exists or the timeout is reached.
func waitForObject(ctx context.Context, client Client, bucket, object string) error {
	deadline := time.After(logObjectWaitTimeout)
	ticker := time.NewTicker(logPollInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timed out waiting for log object gs://%s/%s", bucket, object)
		case <-ticker.C:
			exists, err := client.ObjectExists(ctx, bucket, object)
			if err != nil {
				continue // transient error, keep trying
			}

			if exists {
				return nil
			}
		}
	}
}

// pollLogChunk reads new bytes from the log object starting at offset,
// splits them into lines, and sends each line to the channel.
// Returns the updated offset.
func pollLogChunk(ctx context.Context, client Client, bucket, object string, offset int64, ch chan<- string) (int64, error) {
	size, err := client.ObjectSize(ctx, bucket, object)
	if err != nil {
		return offset, err
	}

	if size <= offset {
		return offset, nil // no new data
	}

	data, err := client.ReadObjectRange(ctx, bucket, object, offset)
	if err != nil {
		return offset, err
	}

	if len(data) == 0 {
		return offset, nil
	}

	// Find the last newline. Only send complete lines; hold back any
	// incomplete trailing fragment for the next poll cycle.
	lastNewline := bytes.LastIndexByte(data, '\n')
	if lastNewline < 0 {
		// No newline at all — all data is an incomplete line.
		return offset, nil
	}

	// Split the complete portion into lines.
	complete := data[:lastNewline]
	lines := bytes.Split(complete, []byte("\n"))

	for _, line := range lines {
		select {
		case ch <- string(line):
		case <-ctx.Done():
			return offset, ctx.Err()
		}
	}

	// Advance offset past the last newline.
	return offset + int64(lastNewline) + 1, nil
}
