package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// runGcloud executes a gcloud command and returns its stdout.
// On non-zero exit, returns an error wrapping stderr.
func runGcloud(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gcloud", args...)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gcloud %s failed: %s: %w",
			strings.Join(args[:min(len(args), 3)], " "),
			strings.TrimSpace(stderr.String()), err)
	}

	return stdout.Bytes(), nil
}

// runGcloudJSON executes a gcloud command with --format=json and unmarshals into target.
func runGcloudJSON(ctx context.Context, target interface{}, args ...string) error {
	output, err := runGcloud(ctx, args...)
	if err != nil {
		return err
	}

	return json.Unmarshal(output, target)
}
