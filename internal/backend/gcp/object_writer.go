package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

// GCSObjectWriter adapts cloud.google.com/go/storage to the logs.ObjectWriter interface.
type GCSObjectWriter struct {
	client *storage.Client
}

// NewGCSObjectWriter creates a new GCSObjectWriter using Application Default Credentials.
func NewGCSObjectWriter(ctx context.Context) (*GCSObjectWriter, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSObjectWriter{client: client}, nil
}

// WriteObject writes data to the given GCS bucket/object, overwriting if it exists.
func (w *GCSObjectWriter) WriteObject(ctx context.Context, bucket, object string, data []byte) error {
	writer := w.client.Bucket(bucket).Object(object).NewWriter(ctx)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return fmt.Errorf("failed to write GCS object: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer: %w", err)
	}

	return nil
}

// Close releases the underlying GCS storage client.
func (w *GCSObjectWriter) Close() error {
	return w.client.Close()
}
