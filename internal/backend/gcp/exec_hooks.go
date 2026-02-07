package gcp

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/compute/metadata"

	"github.com/rickihastings/spinner/internal/exec"
	"github.com/rickihastings/spinner/internal/logs"
)

// gcsEnv holds the resolved GCS environment for a GCE VM.
type gcsEnv struct {
	bucket       string
	instanceName string
	writer       *gcsObjectWriter
}

// resolveGCSEnv checks whether we're on GCE, reads the given bucket env var,
// verifies SPINNER_INSTANCE_NAME, and creates an ObjectWriter. Returns nil if
// any precondition is not met (not on GCE, env var unset, client error).
func resolveGCSEnv(ctx context.Context, bucketEnvVar, feature string) *gcsEnv {
	if !metadata.OnGCE() {
		return nil
	}

	bucket := os.Getenv(bucketEnvVar)
	if bucket == "" {
		return nil
	}

	instanceName := os.Getenv("SPINNER_INSTANCE_NAME")
	if instanceName == "" {
		fmt.Fprintf(os.Stderr, "Warning: %s set but SPINNER_INSTANCE_NAME empty, skipping %s\n", bucketEnvVar, feature)
		return nil
	}

	objectWriter, err := newGCSObjectWriter(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create GCS client for %s: %v\n", feature, err)
		return nil
	}

	return &gcsEnv{bucket: bucket, instanceName: instanceName, writer: objectWriter}
}

// NewLogSinkFactory returns a LogSinkFactory that creates a GCS log sink
// when running on a GCE VM with SPINNER_LOG_BUCKET set.
func NewLogSinkFactory() exec.LogSinkFactory {
	return func(ctx context.Context) (io.Writer, func()) {
		env := resolveGCSEnv(ctx, "SPINNER_LOG_BUCKET", "GCS log sink")
		if env == nil {
			return nil, nil
		}

		object := env.instanceName + "/logs/raw.log"
		sink := logs.NewGCSSink(ctx, env.writer, env.bucket, object)

		fmt.Printf("GCS log streaming enabled: gs://%s/%s\n", env.bucket, object)

		cleanup := func() {
			_ = sink.Close()
			_ = env.writer.Close()
		}

		return sink, cleanup
	}
}

// NewStateSyncFactory returns a StateSyncFactory that creates a GCS state sync
// function when running on a GCE VM with SPINNER_STATE_BUCKET set.
func NewStateSyncFactory() exec.StateSyncFactory {
	return func(ctx context.Context) exec.StateSyncFunc {
		env := resolveGCSEnv(ctx, "SPINNER_STATE_BUCKET", "GCS state sync")
		if env == nil {
			return nil
		}

		object := env.instanceName + "/state.json"
		fmt.Printf("GCS state sync enabled: gs://%s/%s\n", env.bucket, object)

		return func(statePath string) {
			data, readErr := os.ReadFile(statePath)
			if readErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to read state file for GCS sync: %v\n", readErr)
				return
			}

			if writeErr := env.writer.WriteObject(context.Background(), env.bucket, object, data); writeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to sync state to GCS: %v\n", writeErr)
			}
		}
	}
}
