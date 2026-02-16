package gcp

import (
	"context"
)

// gcsObjectWriter adapts the Client interface to the logs.ObjectWriter interface.
type gcsObjectWriter struct {
	client Client
}

// newGCSObjectWriter creates a new gcsObjectWriter using the given Client.
func newGCSObjectWriter(client Client) *gcsObjectWriter {
	return &gcsObjectWriter{client: client}
}

// WriteObject writes data to the given GCS bucket/object, overwriting if it exists.
func (w *gcsObjectWriter) WriteObject(ctx context.Context, bucket, object string, data []byte) error {
	return w.client.WriteObject(ctx, bucket, object, data)
}

// Close is a no-op — no resources to release with CLI-based client.
func (w *gcsObjectWriter) Close() error {
	return nil
}
