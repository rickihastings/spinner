package gcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rickihastings/spinner/internal/util"
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
// Options: "project", "zone", "machine-type", "disk-size", "state-bucket", "bake-script".
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

	// Load custom bake script contents (if path provided)
	customBakeScript, err := loadBakeScriptFile(config.Options["bake-script"])
	if err != nil {
		return err
	}

	// Load and render bake script template with custom script
	bakeScript, err := loadBakeScript(customBakeScript)
	if err != nil {
		return fmt.Errorf("failed to load bake script: %w", err)
	}

	// Load the standard startup.sh so it can be embedded in the baked image.
	// The bake script reads this from metadata and installs it at /usr/local/bin/startup.sh.
	startupScript, err := loadStartupScript()
	if err != nil {
		return fmt.Errorf("failed to load startup script: %w", err)
	}

	// Load the shared install_spinner.sh script
	installScript, err := loadInstallSpinnerScript()
	if err != nil {
		return fmt.Errorf("failed to load install script: %w", err)
	}

	stateBucket := config.Options["state-bucket"]

	// Check if we're in development mode (local binary exists)
	// This happens when running from source after ./scripts/dev-setup.sh
	if stateBucket != "" {
		projectRoot, rootErr := util.FindProjectRoot()
		if rootErr == nil {
			localTarball := filepath.Join(projectRoot, "dist", "spinner-dev-linux-amd64.tar.gz")
			if _, statErr := os.Stat(localTarball); statErr == nil {
				// Local binary exists - upload it automatically
				fmt.Println("🔧 Local development binary detected, uploading to GCS...")

				if err := uploadLocalBinary(ctx, p.client, stateBucket); err != nil {
					return fmt.Errorf("failed to upload local binary: %w", err)
				}

				// Set LOCAL_BUILD so bake VM knows to download from GCS
				err := os.Setenv("LOCAL_BUILD", "true")
				if err != nil {
					return fmt.Errorf("failed to set LOCAL_BUILD env var: %w", err)
				}

				fmt.Println("✓ Local binary uploaded to state bucket")
			}
		}
	}

	// Read config hash from environment if provided (used by integration tests)
	configHash := os.Getenv("SPINNER_CONFIG_HASH")

	return bakeImage(ctx, p.client, bakeConfig{
		ImageName:     config.Name,
		Project:       project,
		Zone:          zone,
		MachineType:   machineType,
		DiskSizeGB:    diskSizeGB,
		StartupScript: bakeScript,
		StateBucket:   stateBucket,
		ConfigHash:    configHash,
		ExtraMetadata: map[string]string{
			"startup-script-runtime": startupScript,
			"spinner-install-script": installScript,
		},
	})
}

// InstanceName returns the deterministic VM instance name for the given config.
// GCP instance names: lowercase, max 63 chars, [a-z]([-a-z0-9]*[a-z0-9])?.
func (p *Provider) InstanceName(config provider.CreateConfig) string {
	image := config.Options["image"]
	return generateInstanceName(image, config.Repo, config.Branch)
}

// Create creates and starts a new VM instance from a baked image.
// Options: "image", "project", "zone", "machine-type", "disk-size", "state-bucket".
func (p *Provider) Create(ctx context.Context, config provider.CreateConfig) (*provider.Instance, error) {
	image := config.Options["image"]
	name := p.InstanceName(config)

	// Verify the baked image exists
	_, err := p.client.GetImage(ctx, p.project, image)
	if err != nil {
		return nil, fmt.Errorf("image '%s' not found in project '%s' — run setup first: %w", image, p.project, err)
	}

	// Load runtime startup script
	runtimeScript, err := loadRuntimeScript()
	if err != nil {
		return nil, fmt.Errorf("failed to load runtime script: %w", err)
	}

	machineType := config.Options["machine-type"]
	if machineType == "" {
		machineType = "e2-standard-2"
	}

	var diskSizeGB int64 = 30

	if ds := config.Options["disk-size"]; ds != "" {
		parsed, parseErr := strconv.ParseInt(ds, 10, 64)
		if parseErr == nil && parsed > 0 {
			diskSizeGB = parsed
		}
	}

	maxIterations := config.MaxIterations
	if maxIterations == "" {
		maxIterations = "100"
	}

	metadata := map[string]string{
		"startup-script":          runtimeScript,
		"REPO_URL":                config.Repo,
		"PROMPT":                  config.Prompt,
		"BRANCH":                  config.Branch,
		"MAX_ITERATIONS":          maxIterations,
		"GITHUB_TOKEN":            os.Getenv("GITHUB_TOKEN"),
		"CLAUDE_CODE_OAUTH_TOKEN": os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"),
		"SPINNER_INSTANCE_NAME":   name,
	}

	if p.bucket != "" {
		metadata["SPINNER_LOG_BUCKET"] = p.bucket
		metadata["SPINNER_STATE_BUCKET"] = p.bucket
	}

	// Add custom env vars with SPINNER_ENV_ prefix
	for key, value := range config.EnvVars {
		metadata["SPINNER_ENV_"+key] = value
	}

	labels := map[string]string{
		"spinner-managed": "true",
		"spinner-image":   sanitizeLabel(image),
		"spinner-repo":    sanitizeLabel(extractRepoName(config.Repo)),
	}

	err = p.client.CreateInstance(ctx, instanceConfig{
		Name:         name,
		Project:      p.project,
		Zone:         p.zone,
		MachineType:  machineType,
		ImageProject: p.project,
		ImageName:    image,
		DiskSizeGB:   diskSizeGB,
		Network:      "default",
		ExternalIP:   true,
		Metadata:     metadata,
		Labels:       labels,
		Scopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_write",
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring.write",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	return &provider.Instance{
		Name:   name,
		Status: provider.InstanceStatusRunning,
	}, nil
}

// Start starts a stopped VM instance.
func (p *Provider) Start(ctx context.Context, name string) (*provider.Instance, error) {
	if err := p.client.StartInstance(ctx, p.project, p.zone, name); err != nil {
		return nil, fmt.Errorf("failed to start instance: %w", err)
	}

	return &provider.Instance{
		Name:   name,
		Status: provider.InstanceStatusRunning,
	}, nil
}

// Restart stops then starts a VM instance.
func (p *Provider) Restart(ctx context.Context, name string) (*provider.Instance, error) {
	if err := p.Stop(ctx, name); err != nil {
		return nil, err
	}

	return p.Start(ctx, name)
}

// Stop stops a running VM instance.
func (p *Provider) Stop(ctx context.Context, name string) error {
	return p.client.StopInstance(ctx, p.project, p.zone, name)
}

// Remove deletes a VM instance and its boot disk.
func (p *Provider) Remove(ctx context.Context, name string) error {
	return p.client.DeleteInstance(ctx, p.project, p.zone, name)
}

// Logs returns the VM's full log output from GCS.
func (p *Provider) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	if p.bucket == "" {
		return nil, fmt.Errorf("gcp: no state bucket configured; cannot read logs")
	}

	return readLogs(ctx, p.client, p.bucket, name)
}

// Status returns the current lifecycle status of a VM instance.
func (p *Provider) Status(ctx context.Context, name string) (provider.InstanceStatus, error) {
	instance, err := p.client.GetInstance(ctx, p.project, p.zone, name)
	if err != nil {
		// If the instance doesn't exist, return None (not an error).
		// GCP SDK wraps 404s in an error; check the message.
		if isNotFoundError(err) {
			return provider.InstanceStatusNone, nil
		}

		return provider.InstanceStatusNone, fmt.Errorf("failed to get instance status: %w", err)
	}

	return mapVMStatus(instance.GetStatus()), nil
}

// WatchLogs streams log lines from GCS by polling for new content.
func (p *Provider) WatchLogs(ctx context.Context, name string, _ int, ch chan<- string) error {
	if p.bucket == "" {
		return fmt.Errorf("gcp: no state bucket configured; cannot stream logs")
	}

	return watchLogs(ctx, p.client, p.bucket, name, ch)
}

// WatchMetrics streams resource metrics from Cloud Monitoring.
// Polls CPU utilization at 60-second intervals and maps VM state to ContainerMetrics.
func (p *Provider) WatchMetrics(ctx context.Context, name string, ch chan<- provider.ContainerMetrics) error {
	return streamGCPMetrics(ctx, p.client, p.project, p.zone, name, ch)
}

// GetInstanceMetadata returns metadata about a GCP VM instance.
func (p *Provider) GetInstanceMetadata(ctx context.Context, name string) (*provider.InstanceMetadata, error) {
	instance, err := p.client.GetInstance(ctx, p.project, p.zone, name)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("instance not found: %s", name)
		}

		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	metadata := &provider.InstanceMetadata{
		Backend:    "gcp",
		InstanceID: name, // For GCP, the instance name is the identifier
	}

	// Extract boot disk name as the "image" identifier
	if len(instance.Disks) > 0 && instance.Disks[0] != nil {
		// The boot disk source is a full URL like:
		// https://www.googleapis.com/compute/v1/projects/PROJECT/zones/ZONE/disks/DISK_NAME
		// Extract just the disk name
		diskSource := instance.Disks[0].GetSource()
		if diskSource != "" {
			// Get the last part of the URL path
			parts := strings.Split(diskSource, "/")
			if len(parts) > 0 {
				metadata.ImageID = parts[len(parts)-1]
			}
		}
	}

	// Try to get agent and max iterations from instance metadata
	if instance.Metadata != nil && instance.Metadata.Items != nil {
		for _, item := range instance.Metadata.Items {
			if item == nil {
				continue
			}

			key := item.GetKey()
			value := item.GetValue()

			switch key {
			case "ANTHROPIC_MODEL":
				if value != "" {
					metadata.Agent = value
				}
			case "MAX_ITERATIONS":
				if value != "" {
					if val, parseErr := fmt.Sscanf(value, "%d", &metadata.MaxIterations); parseErr != nil || val != 1 {
						_ = fmt.Errorf("failed to parse max iterations: %s", value)
					}
				}
			case "BRANCH":
				if value != "" {
					metadata.Branch = value
				}
			}
		}
	}

	return metadata, nil
}

// isNotFoundError checks whether a GCP API error indicates a resource was not found.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "notfound") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404")
}
