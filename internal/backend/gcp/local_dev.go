package gcp

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rickihastings/spinner/internal/util"
	"github.com/rickihastings/spinner/internal/version"
)

// uploadLocalBinary uploads the locally-built spinner binary to the state bucket
// for use during image baking. Only called for non-release (dev) builds.
func uploadLocalBinary(ctx context.Context, client Client, bucket string) error {
	// Try 1: find the tarball from the project source tree
	projectRoot, err := util.FindProjectRoot()
	if err == nil {
		tarballPath := filepath.Join(projectRoot, "dist", "spinner-dev-linux-amd64.tar.gz")

		data, readErr := os.ReadFile(tarballPath)
		if readErr == nil {
			return uploadTarball(ctx, client, bucket, data)
		}
	}

	// Try 2: dev build running on Linux — create tarball from the running binary
	if !version.IsRelease() && runtime.GOOS == "linux" {
		execPath, execErr := os.Executable()
		if execErr == nil {
			data, tarErr := createTarballFromBinary(execPath)
			if tarErr != nil {
				return fmt.Errorf("failed to create tarball from running binary: %w", tarErr)
			}

			return uploadTarball(ctx, client, bucket, data)
		}
	}

	return fmt.Errorf("failed to find spinner binary for upload\n" +
		"Run 'make build' from the source tree, or run a dev build directly on Linux")
}

// uploadTarball uploads the given tarball data to the state bucket.
func uploadTarball(ctx context.Context, client Client, bucket string, data []byte) error {
	objectPath := "local-dev/spinner-dev-linux-amd64.tar.gz"
	if err := client.WriteObject(ctx, bucket, objectPath, data); err != nil {
		return fmt.Errorf("failed to upload to gs://%s/%s: %w", bucket, objectPath, err)
	}

	return nil
}

// createTarballFromBinary creates a .tar.gz containing the given binary as "spinner".
func createTarballFromBinary(binaryPath string) ([]byte, error) {
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary: %w", err)
	}

	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "spinner",
		Mode: 0755,
		Size: int64(len(binaryData)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return nil, fmt.Errorf("failed to write tar header: %w", err)
	}

	if _, err := tw.Write(binaryData); err != nil {
		return nil, fmt.Errorf("failed to write tar body: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
