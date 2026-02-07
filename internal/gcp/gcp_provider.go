package gcp

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/rickihastings/spinner/internal/provider"
)

// Provider implements provider.Provider using GCP Compute Engine VMs as the backend.
type Provider struct {
	client  Client
	project string
	zone    string
	bucket  string
}

// NewGCPProvider creates a new Provider backed by the given Client.
func NewGCPProvider(client Client, project, zone, bucket string) *Provider {
	return &Provider{
		client:  client,
		project: project,
		zone:    zone,
		bucket:  bucket,
	}
}

// Setup provisions a named environment by baking a GCP Compute Engine image.
// Options: "project", "zone", "machine-type", "disk-size", "state-bucket".
func (p *Provider) Setup(ctx context.Context, config provider.SetupConfig) error {
	project := config.Options["project"]
	zone := config.Options["zone"]

	machineType := config.Options["machine-type"]
	if machineType == "" {
		machineType = "e2-standard-2"
	}

	var diskSizeGB int64 = 30

	if ds := config.Options["disk-size"]; ds != "" {
		parsed, err := strconv.ParseInt(ds, 10, 64)
		if err == nil && parsed > 0 {
			diskSizeGB = parsed
		}
	}

	// Check if image already exists
	_, err := p.client.GetImage(ctx, project, config.Name)
	if err == nil {
		fmt.Printf("Image '%s' already exists, deleting before rebuild...\n", config.Name)

		if deleteErr := p.client.DeleteImage(ctx, project, config.Name); deleteErr != nil {
			return fmt.Errorf("failed to delete existing image: %w", deleteErr)
		}
	}

	// Load bake script
	bakeScript, err := LoadBakeScript()
	if err != nil {
		return fmt.Errorf("failed to load bake script: %w", err)
	}

	return BakeImage(ctx, p.client, BakeConfig{
		ImageName:     config.Name,
		Project:       project,
		Zone:          zone,
		MachineType:   machineType,
		DiskSizeGB:    diskSizeGB,
		StartupScript: bakeScript,
	})
}

// InstanceName returns the deterministic VM instance name for the given config.
// GCP instance names: lowercase, max 63 chars, [a-z]([-a-z0-9]*[a-z0-9])?.
func (p *Provider) InstanceName(_ provider.CreateConfig) string {
	return ""
}

// Create creates and starts a new VM instance from a baked image.
func (p *Provider) Create(_ context.Context, _ provider.CreateConfig) (*provider.Instance, error) {
	return nil, fmt.Errorf("gcp: Create not yet implemented (see slice 4.0)")
}

// Start starts a stopped VM instance.
func (p *Provider) Start(_ context.Context, _ string) (*provider.Instance, error) {
	return nil, fmt.Errorf("gcp: Start not yet implemented (see slice 4.0)")
}

// Restart stops then starts a VM instance.
func (p *Provider) Restart(_ context.Context, _ string) (*provider.Instance, error) {
	return nil, fmt.Errorf("gcp: Restart not yet implemented (see slice 4.0)")
}

// Stop stops a running VM instance.
func (p *Provider) Stop(_ context.Context, _ string) error {
	return fmt.Errorf("gcp: Stop not yet implemented (see slice 4.0)")
}

// Remove deletes a VM instance.
func (p *Provider) Remove(_ context.Context, _ string) error {
	return fmt.Errorf("gcp: Remove not yet implemented (see slice 4.0)")
}

// Logs returns the VM's log output from GCS.
func (p *Provider) Logs(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("gcp: Logs not yet implemented (see slice 5.0)")
}

// Status returns the current lifecycle status of a VM instance.
func (p *Provider) Status(_ context.Context, _ string) (provider.InstanceStatus, error) {
	return provider.InstanceStatusNone, fmt.Errorf("gcp: Status not yet implemented (see slice 4.0)")
}

// WatchLogs streams log lines from GCS.
func (p *Provider) WatchLogs(_ context.Context, _ string, _ int, _ chan<- string) error {
	return fmt.Errorf("gcp: WatchLogs not yet implemented (see slice 5.0)")
}

// WatchMetrics streams resource metrics from Cloud Monitoring.
func (p *Provider) WatchMetrics(_ context.Context, _ string, _ chan<- provider.ContainerMetrics) error {
	return fmt.Errorf("gcp: WatchMetrics not yet implemented (see slice 5.5)")
}
