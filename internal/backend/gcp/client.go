package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Client defines the interface for GCP operations.
// This interface enables dependency injection and mocking for testability,
// mirroring the Docker backend's two-layer pattern (Provider -> Client).
type Client interface {
	// Instance operations
	CreateInstance(ctx context.Context, config instanceConfig) error
	GetInstance(ctx context.Context, project, zone, name string) (*GCPInstance, error)
	SetMetadata(ctx context.Context, project, zone, name string, metadata *GCPMetadata) error
	StartInstance(ctx context.Context, project, zone, name string) error
	StopInstance(ctx context.Context, project, zone, name string) error
	ResetInstance(ctx context.Context, project, zone, name string) error
	DeleteInstance(ctx context.Context, project, zone, name string) error

	// Image operations
	CreateImage(ctx context.Context, project string, config imageConfig) error
	GetImage(ctx context.Context, project, name string) (*GCPImage, error)
	DeleteImage(ctx context.Context, project, name string) error

	// ListInstances lists VM instances matching a label filter.
	ListInstances(ctx context.Context, project, zone string, filter string) ([]*GCPInstance, error)

	// Serial port (for boot/bake diagnostics)
	GetSerialPortOutput(ctx context.Context, project, zone, name string, start int64) (*serialPortOutput, error)

	// Storage operations (GCS for state persistence + log streaming)
	WriteObject(ctx context.Context, bucket, object string, data []byte) error
	ReadObject(ctx context.Context, bucket, object string) ([]byte, error)
	ReadObjectRange(ctx context.Context, bucket, object string, offset int64) ([]byte, error)
	ObjectSize(ctx context.Context, bucket, object string) (int64, error)
	ObjectExists(ctx context.Context, bucket, object string) (bool, error)
	DeleteObjectsWithPrefix(ctx context.Context, bucket, prefix string) error

	// Close releases resources (no-op for CLI-based client).
	Close() error
}

// RealGCPClient implements Client using the gcloud CLI.
// Authentication is delegated to gcloud's own credential management.
type RealGCPClient struct{}

// checkGcloudInstalled verifies that the gcloud CLI is available on PATH.
func checkGcloudInstalled() error {
	if _, err := exec.LookPath("gcloud"); err != nil {
		return fmt.Errorf("gcloud CLI not found on PATH: install from https://cloud.google.com/sdk/docs/install")
	}

	return nil
}

// NewRealGCPClient creates a new RealGCPClient after verifying that the gcloud
// CLI is available on PATH.
func NewRealGCPClient(_ context.Context) (*RealGCPClient, error) {
	if err := checkGcloudInstalled(); err != nil {
		return nil, err
	}

	return &RealGCPClient{}, nil
}

// Close is a no-op for the CLI-based client (no SDK resources to release).
func (c *RealGCPClient) Close() error {
	return nil
}

// CreateInstance creates a GCP Compute Engine VM instance using gcloud CLI.
func (c *RealGCPClient) CreateInstance(ctx context.Context, config instanceConfig) error {
	args := []string{
		"compute", "instances", "create", config.Name,
		"--project=" + config.Project,
		"--zone=" + config.Zone,
		"--machine-type=" + config.MachineType,
		"--image=" + config.ImageName,
		"--image-project=" + config.ImageProject,
		"--boot-disk-size=" + strconv.FormatInt(config.DiskSizeGB, 10) + "GB",
		"--network=" + config.Network,
		"--quiet",
		"--format=json",
	}

	diskType := config.DiskType
	if diskType == "" {
		diskType = "pd-balanced"
	}

	args = append(args, "--boot-disk-type="+diskType)

	if config.Subnet != "" {
		args = append(args, "--subnet="+config.Subnet)
	}

	if !config.ExternalIP {
		args = append(args, "--no-address")
	}

	if len(config.Metadata) > 0 {
		var pairs []string
		for k, v := range config.Metadata {
			pairs = append(pairs, k+"="+v)
		}

		args = append(args, "--metadata="+strings.Join(pairs, ","))
	}

	if len(config.Labels) > 0 {
		var pairs []string
		for k, v := range config.Labels {
			pairs = append(pairs, k+"="+v)
		}

		args = append(args, "--labels="+strings.Join(pairs, ","))
	}

	if config.ServiceAccount != "" || len(config.Scopes) > 0 {
		email := config.ServiceAccount
		if email == "" {
			email = "default"
		}

		args = append(args, "--service-account="+email)

		if len(config.Scopes) > 0 {
			args = append(args, "--scopes="+strings.Join(config.Scopes, ","))
		}
	}

	_, err := runGcloud(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return nil
}

// GetInstance retrieves information about a VM instance.
func (c *RealGCPClient) GetInstance(ctx context.Context, project, zone, name string) (*GCPInstance, error) {
	var instance GCPInstance

	err := runGcloudJSON(ctx, &instance,
		"compute", "instances", "describe", name,
		"--project="+project,
		"--zone="+zone,
		"--format=json",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &instance, nil
}

// SetMetadata sets metadata on a VM instance using gcloud compute instances add-metadata.
// gcloud handles fingerprint conflicts internally with retries.
func (c *RealGCPClient) SetMetadata(ctx context.Context, project, zone, name string, metadata *GCPMetadata) error {
	if len(metadata.Items) == 0 {
		return nil
	}

	var pairs []string
	for _, item := range metadata.Items {
		pairs = append(pairs, item.Key+"="+item.Value)
	}

	_, err := runGcloud(ctx,
		"compute", "instances", "add-metadata", name,
		"--project="+project,
		"--zone="+zone,
		"--metadata="+strings.Join(pairs, ","),
	)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

// StartInstance starts a stopped VM instance.
func (c *RealGCPClient) StartInstance(ctx context.Context, project, zone, name string) error {
	_, err := runGcloud(ctx,
		"compute", "instances", "start", name,
		"--project="+project,
		"--zone="+zone,
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return nil
}

// StopInstance stops a running VM instance.
func (c *RealGCPClient) StopInstance(ctx context.Context, project, zone, name string) error {
	_, err := runGcloud(ctx,
		"compute", "instances", "stop", name,
		"--project="+project,
		"--zone="+zone,
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}

// ResetInstance resets a VM instance (hard reboot).
func (c *RealGCPClient) ResetInstance(ctx context.Context, project, zone, name string) error {
	_, err := runGcloud(ctx,
		"compute", "instances", "reset", name,
		"--project="+project,
		"--zone="+zone,
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("failed to reset instance: %w", err)
	}

	return nil
}

// DeleteInstance deletes a VM instance.
func (c *RealGCPClient) DeleteInstance(ctx context.Context, project, zone, name string) error {
	_, err := runGcloud(ctx,
		"compute", "instances", "delete", name,
		"--project="+project,
		"--zone="+zone,
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

// ListInstances lists VM instances matching a label filter expression.
func (c *RealGCPClient) ListInstances(ctx context.Context, project, zone string, filter string) ([]*GCPInstance, error) {
	args := []string{
		"compute", "instances", "list",
		"--project=" + project,
		"--zones=" + zone,
		"--format=json",
	}

	if filter != "" {
		args = append(args, "--filter="+filter)
	}

	var instances []*GCPInstance

	err := runGcloudJSON(ctx, &instances, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return instances, nil
}

// CreateImage creates a Compute Engine image from a source disk.
func (c *RealGCPClient) CreateImage(ctx context.Context, project string, config imageConfig) error {
	args := []string{
		"compute", "images", "create", config.Name,
		"--project=" + project,
		"--source-disk=" + config.SourceDisk,
		"--quiet",
		"--format=json",
	}

	// Extract zone from source disk URL if present
	// Format: projects/PROJECT/zones/ZONE/disks/DISK_NAME
	if parts := strings.Split(config.SourceDisk, "/"); len(parts) >= 4 {
		for i, p := range parts {
			if p == "zones" && i+1 < len(parts) {
				args = append(args, "--source-disk-zone="+parts[i+1])

				break
			}
		}
	}

	if len(config.Labels) > 0 {
		var pairs []string
		for k, v := range config.Labels {
			pairs = append(pairs, k+"="+v)
		}

		args = append(args, "--labels="+strings.Join(pairs, ","))
	}

	if config.Description != "" {
		args = append(args, "--description="+config.Description)
	}

	_, err := runGcloud(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	return nil
}

// GetImage retrieves information about a Compute Engine image.
func (c *RealGCPClient) GetImage(ctx context.Context, project, name string) (*GCPImage, error) {
	var image GCPImage

	err := runGcloudJSON(ctx, &image,
		"compute", "images", "describe", name,
		"--project="+project,
		"--format=json",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return &image, nil
}

// DeleteImage deletes a Compute Engine image.
func (c *RealGCPClient) DeleteImage(ctx context.Context, project, name string) error {
	_, err := runGcloud(ctx,
		"compute", "images", "delete", name,
		"--project="+project,
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

// GetSerialPortOutput reads serial port output from a VM instance.
func (c *RealGCPClient) GetSerialPortOutput(ctx context.Context, project, zone, name string, start int64) (*serialPortOutput, error) {
	output, err := runGcloud(ctx,
		"compute", "instances", "get-serial-port-output", name,
		"--project="+project,
		"--zone="+zone,
		"--start="+strconv.FormatInt(start, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get serial port output: %w", err)
	}

	// gcloud get-serial-port-output writes content to stdout and the next byte offset
	// as the last line in the format: "Specify --start=N to continue reading"
	content := string(output)

	var next int64

	if idx := strings.LastIndex(content, "\n--\nSpecify --start="); idx != -1 {
		// Parse next offset from the trailer line
		trailer := content[idx:]
		content = content[:idx]

		// Extract number from "Specify --start=N to continue reading"
		if start := strings.Index(trailer, "--start="); start != -1 {
			numStr := trailer[start+len("--start="):]
			if end := strings.IndexByte(numStr, ' '); end != -1 {
				numStr = numStr[:end]
			}

			next, _ = strconv.ParseInt(numStr, 10, 64)
		}
	}

	return &serialPortOutput{
		Contents: content,
		Next:     next,
	}, nil
}

// WriteObject writes data to a GCS object using gcloud storage cp with stdin.
func (c *RealGCPClient) WriteObject(ctx context.Context, bucket, object string, data []byte) error {
	gcsPath := fmt.Sprintf("gs://%s/%s", bucket, object)

	cmd := exec.CommandContext(ctx, "gcloud", "storage", "cp", "-", gcsPath)
	cmd.Stdin = bytes.NewReader(data)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write object: %s: %w",
			strings.TrimSpace(stderr.String()), err)
	}

	return nil
}

// ReadObject reads the full contents of a GCS object.
func (c *RealGCPClient) ReadObject(ctx context.Context, bucket, object string) ([]byte, error) {
	gcsPath := fmt.Sprintf("gs://%s/%s", bucket, object)

	output, err := runGcloud(ctx, "storage", "cat", gcsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return output, nil
}

// ReadObjectRange reads a GCS object starting from the given byte offset.
// Reads the full object and slices from offset in Go (sufficient for small state files).
func (c *RealGCPClient) ReadObjectRange(ctx context.Context, bucket, object string, offset int64) ([]byte, error) {
	data, err := c.ReadObject(ctx, bucket, object)
	if err != nil {
		return nil, err
	}

	if offset >= int64(len(data)) {
		return []byte{}, nil
	}

	return data[offset:], nil
}

// gcsObjectInfo represents a single object in gcloud storage ls --format=json output.
type gcsObjectInfo struct {
	Size json.Number `json:"size"`
}

// ObjectSize returns the size of a GCS object in bytes.
func (c *RealGCPClient) ObjectSize(ctx context.Context, bucket, object string) (int64, error) {
	gcsPath := fmt.Sprintf("gs://%s/%s", bucket, object)

	var objects []gcsObjectInfo

	err := runGcloudJSON(ctx, &objects, "storage", "ls", "-l", gcsPath, "--format=json")
	if err != nil {
		return 0, fmt.Errorf("failed to get object size: %w", err)
	}

	if len(objects) == 0 {
		return 0, fmt.Errorf("object not found: %s", gcsPath)
	}

	size, err := objects[0].Size.Int64()
	if err != nil {
		return 0, fmt.Errorf("failed to parse object size: %w", err)
	}

	return size, nil
}

// ObjectExists checks whether a GCS object exists.
func (c *RealGCPClient) ObjectExists(ctx context.Context, bucket, object string) (bool, error) {
	gcsPath := fmt.Sprintf("gs://%s/%s", bucket, object)

	_, err := runGcloud(ctx, "storage", "ls", gcsPath)
	if err != nil {
		// gcloud storage ls exits non-zero when object doesn't exist
		if strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "NotFound") ||
			strings.Contains(err.Error(), "CommandException") {
			return false, nil
		}

		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// DeleteObjectsWithPrefix deletes all objects in a bucket with the given prefix.
func (c *RealGCPClient) DeleteObjectsWithPrefix(ctx context.Context, bucket, prefix string) error {
	gcsPath := fmt.Sprintf("gs://%s/%s**", bucket, prefix)

	_, err := runGcloud(ctx, "storage", "rm", gcsPath)
	if err != nil {
		// Ignore "not found" errors — no objects to delete is not an error
		if strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "NotFound") ||
			strings.Contains(err.Error(), "CommandException") {
			return nil
		}

		return fmt.Errorf("failed to delete objects with prefix: %w", err)
	}

	return nil
}
