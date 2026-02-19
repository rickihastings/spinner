package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rickihastings/spinner/internal/util"
)

const (
	// defaultMachineType is the default GCP machine type for instances.
	// Users can override via --provider-args="--machine-type=n2-standard-4".
	defaultMachineType = "e2-standard-2"

	// defaultDiskSizeGB is the default boot disk size in GB.
	// Users can override via --provider-args="--boot-disk-size=50GB".
	defaultDiskSizeGB int64 = 30
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
// Options: "project", "zone", "state-bucket", "bake-script".
// ProviderArgs are forwarded to gcloud compute instances create for the bake VM.
func (p *Provider) Setup(ctx context.Context, config provider.SetupConfig) error {
	project := config.Options["project"]
	zone := config.Options["zone"]

	// Validate provider args don't conflict with Spinner-managed flags
	if err := ValidateGCPInstanceArgs(config.ProviderArgs); err != nil {
		return err
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
	startupScript := util.LoadStartupScript()

	// Load the shared install_spinner.sh script
	installScript := util.LoadInstallSpinnerScript()

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
		MachineType:   defaultMachineType,
		DiskSizeGB:    defaultDiskSizeGB,
		StartupScript: bakeScript,
		StateBucket:   stateBucket,
		ConfigHash:    configHash,
		ExtraMetadata: map[string]string{
			"startup-script-runtime": startupScript,
			"spinner-install-script": installScript,
		},
		ExtraArgs: config.ProviderArgs,
	})
}

// InstanceName returns the deterministic VM instance name for the given config.
// GCP instance names: lowercase, max 63 chars, [a-z]([-a-z0-9]*[a-z0-9])?.
func (p *Provider) InstanceName(config provider.CreateConfig) string {
	image := config.Options["image"]
	return generateInstanceName(image, config.Repo, config.Branch)
}

// Create creates and starts a new VM instance from a baked image.
// Options: "image". ProviderArgs are forwarded to gcloud compute instances create.
func (p *Provider) Create(ctx context.Context, config provider.CreateConfig) (*provider.Instance, error) {
	// Validate provider args don't conflict with Spinner-managed flags
	if err := ValidateGCPInstanceArgs(config.ProviderArgs); err != nil {
		return nil, err
	}

	image := config.Options["image"]
	name := p.InstanceName(config)

	// Verify the baked image exists
	_, err := p.client.GetImage(ctx, p.project, image)
	if err != nil {
		return nil, fmt.Errorf("image '%s' not found in project '%s' — run setup first: %w", image, p.project, err)
	}

	// Load runtime startup script
	runtimeScript := loadRuntimeScript()

	maxIterations := config.MaxIterations
	if maxIterations == "" {
		maxIterations = "100"
	}

	metadata := map[string]string{
		"startup-script":        runtimeScript,
		"REPO_URL":              config.Repo,
		"PROMPT":                config.Prompt,
		"BRANCH":                config.Branch,
		"MAX_ITERATIONS":        maxIterations,
		"SPINNER_INSTANCE_NAME": name,
		"ANTHROPIC_MODEL":       config.Model,
		"GIT_USER_NAME":         gcpGitConfigValue("user.name"),
		"GIT_USER_EMAIL":        gcpGitConfigValue("user.email"),
	}

	// Tokens travel via encrypted blob, not as plaintext metadata.
	// Base64-encode the blob for metadata transport.
	if len(config.SecretBlob) > 0 {
		metadata["SPINNER_SECRET_BLOB"] = base64.StdEncoding.EncodeToString(config.SecretBlob)
	}

	// Upload ephemeral key to GCS (not metadata — metadata is visible in GCP Console).
	// The key is fetched by gcp_runtime.sh on boot and written to /run/spinner/secrets.key.
	if len(config.SecretKey) > 0 && p.bucket != "" {
		keyObject := name + "/secrets.key"
		if err := p.client.WriteObject(ctx, p.bucket, keyObject, config.SecretKey); err != nil {
			return nil, fmt.Errorf("failed to upload secrets key to GCS: %w", err)
		}
	}

	if p.bucket != "" {
		metadata["SPINNER_LOG_BUCKET"] = p.bucket
		metadata["SPINNER_STATE_BUCKET"] = p.bucket
	}

	// Add custom env vars with SPINNER_ENV_ prefix
	for key, value := range config.EnvVars {
		metadata["SPINNER_ENV_"+key] = value
	}

	// If env file is specified, read and base64-encode it for metadata transport
	if config.EnvFile != "" {
		envFileContent, readErr := os.ReadFile(config.EnvFile)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read env file: %w", readErr)
		}

		metadata["SPINNER_ENV_FILE"] = base64.StdEncoding.EncodeToString(envFileContent)
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
		MachineType:  defaultMachineType,
		ImageProject: p.project,
		ImageName:    image,
		DiskSizeGB:   defaultDiskSizeGB,
		Network:      "default",
		ExternalIP:   true,
		Metadata:     metadata,
		Labels:       labels,
		Scopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_write",
			"https://www.googleapis.com/auth/logging.write",
		},
		ExtraArgs: config.ProviderArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	return &provider.Instance{
		Name:   name,
		Status: provider.InstanceStatusRunning,
	}, nil
}

// Start starts a stopped VM instance, updating its metadata (e.g. prompt)
// from the provided CreateConfig before starting.
func (p *Provider) Start(ctx context.Context, name string, config provider.CreateConfig) (*provider.Instance, error) {
	// Update instance metadata with the latest config before starting.
	// This ensures the prompt and other settings are current when the VM boots.
	if err := p.updateMetadata(ctx, name, config); err != nil {
		return nil, fmt.Errorf("failed to update instance metadata: %w", err)
	}

	if err := p.client.StartInstance(ctx, p.project, p.zone, name); err != nil {
		return nil, fmt.Errorf("failed to start instance: %w", err)
	}

	return &provider.Instance{
		Name:   name,
		Status: provider.InstanceStatusRunning,
	}, nil
}

// updateMetadata fetches current instance metadata, updates config-driven fields,
// and writes the updated metadata back using the fingerprint for consistency.
func (p *Provider) updateMetadata(ctx context.Context, name string, config provider.CreateConfig) error {
	instance, err := p.client.GetInstance(ctx, p.project, p.zone, name)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	metadata := instance.Metadata
	if metadata == nil {
		return nil
	}

	// Build a map of keys to update.
	// Tokens travel via encrypted blob, not as plaintext metadata.
	updates := map[string]string{
		"PROMPT": config.Prompt,
	}

	// Update blob if provided (e.g. secrets may have changed)
	if len(config.SecretBlob) > 0 {
		updates["SPINNER_SECRET_BLOB"] = base64.StdEncoding.EncodeToString(config.SecretBlob)
	}

	// Re-upload key to GCS if provided
	if len(config.SecretKey) > 0 && p.bucket != "" {
		keyObject := name + "/secrets.key"
		if err := p.client.WriteObject(ctx, p.bucket, keyObject, config.SecretKey); err != nil {
			return fmt.Errorf("failed to upload secrets key to GCS: %w", err)
		}
	}

	if config.MaxIterations != "" {
		updates["MAX_ITERATIONS"] = config.MaxIterations
	}

	if config.Model != "" {
		updates["ANTHROPIC_MODEL"] = config.Model
	}

	// Update existing metadata items in-place, removing SPINNER_SECRET_PASSPHRASE
	// (replaced by key-file transport via GCS)
	var filtered []GCPMetadataItem

	for i := range metadata.Items {
		key := metadata.Items[i].Key

		// Remove deprecated passphrase metadata
		if key == "SPINNER_SECRET_PASSPHRASE" {
			continue
		}

		if newVal, ok := updates[key]; ok {
			metadata.Items[i].Value = newVal

			delete(updates, key)
		}

		filtered = append(filtered, metadata.Items[i])
	}

	metadata.Items = filtered

	// Append any new keys that didn't exist in the original metadata
	for key, value := range updates {
		metadata.Items = append(metadata.Items, GCPMetadataItem{
			Key:   key,
			Value: value,
		})
	}

	return p.client.SetMetadata(ctx, p.project, p.zone, name, metadata)
}

// Stop stops a running VM instance.
func (p *Provider) Stop(ctx context.Context, name string) error {
	return p.client.StopInstance(ctx, p.project, p.zone, name)
}

// Remove deletes a VM instance, its boot disk, GCS state, and local cache.
func (p *Provider) Remove(ctx context.Context, name string) error {
	// Delete the VM instance and boot disk
	if err := p.client.DeleteInstance(ctx, p.project, p.zone, name); err != nil {
		return err
	}

	// Clean up GCS bucket state (logs, state.json, etc.)
	if p.bucket != "" {
		prefix := name + "/"
		if err := p.client.DeleteObjectsWithPrefix(ctx, p.bucket, prefix); err != nil {
			return fmt.Errorf("failed to clean up GCS state: %w", err)
		}
	}

	// Clean up local state cache
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".spinner", name)
	if err := os.RemoveAll(stateDir); err != nil {
		return fmt.Errorf("failed to remove local state directory: %w", err)
	}

	return nil
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
		if isNotFoundError(err) {
			return provider.InstanceStatusNone, nil
		}

		return provider.InstanceStatusNone, fmt.Errorf("failed to get instance status: %w", err)
	}

	return mapVMStatus(instance.Status), nil
}

// WatchLogs streams log lines from GCS by polling for new content.
func (p *Provider) WatchLogs(ctx context.Context, name string, _ int, ch chan<- string) error {
	if p.bucket == "" {
		return fmt.Errorf("gcp: no state bucket configured; cannot stream logs")
	}

	return watchLogs(ctx, p.client, p.bucket, name, ch)
}

// WatchMetrics streams resource metrics by polling the GCS state file.
// CPU and memory metrics are written to state.json by the exec loop inside the VM.
func (p *Provider) WatchMetrics(ctx context.Context, name string, ch chan<- provider.ContainerMetrics) error {
	return streamGCPMetrics(ctx, p.client, p.project, p.zone, name, p.bucket, ch)
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

	// Extract boot disk name as the "image" identifier.
	// The boot disk source is a full URL like:
	// https://www.googleapis.com/compute/v1/projects/PROJECT/zones/ZONE/disks/DISK_NAME
	if len(instance.Disks) > 0 {
		diskSource := instance.Disks[0].Source
		if diskSource != "" {
			parts := strings.Split(diskSource, "/")
			metadata.ImageID = parts[len(parts)-1]
		}
	}

	vm := parseVMMetadata(instance.Metadata)
	metadata.Agent = vm.Agent
	metadata.MaxIterations = vm.MaxIterations
	metadata.Branch = vm.Branch

	return metadata, nil
}

// List discovers all spinner-managed GCP instances using label-based filtering.
// Enriches each instance with metadata from VM labels/metadata items and GCS state.
func (p *Provider) List(ctx context.Context) ([]provider.InstanceInfo, error) {
	instances, err := p.client.ListInstances(ctx, p.project, p.zone, "labels.spinner-managed=true")
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var result []provider.InstanceInfo

	for _, inst := range instances {
		name := inst.Name
		status := mapVMStatus(inst.Status)

		info := provider.InstanceInfo{
			Name:    name,
			Status:  status,
			Backend: "gcp",
		}

		// Extract info from VM labels
		if image, ok := inst.Labels["spinner-image"]; ok {
			info.Image = image
		}

		if repo, ok := inst.Labels["spinner-repo"]; ok {
			info.Repo = repo
		}

		// Extract info from VM metadata items
		vm := parseVMMetadata(inst.Metadata)
		info.Agent = vm.Agent
		info.MaxIterations = vm.MaxIterations
		info.Branch = vm.Branch

		// Enrich from GCS state file if bucket is configured
		p.enrichFromGCSState(ctx, &info, name)

		result = append(result, info)
	}

	return result, nil
}

// enrichFromGCSState reads the state.json from GCS and populates execution state
// fields on the InstanceInfo. Silently skips if no bucket is configured or state
// cannot be read.
func (p *Provider) enrichFromGCSState(ctx context.Context, info *provider.InstanceInfo, instanceName string) {
	if p.bucket == "" {
		return
	}

	data, err := readState(ctx, p.client, p.bucket, instanceName)
	if err != nil || data == nil {
		return
	}

	provider.EnrichFromStateData(info, data)
}

// vmMetadata holds parsed values from a VM's metadata items.
type vmMetadata struct {
	Agent         string
	MaxIterations int
	Branch        string
}

// parseVMMetadata extracts known metadata fields from a GCP instance's metadata items.
func parseVMMetadata(metadata *GCPMetadata) vmMetadata {
	var m vmMetadata

	if metadata == nil {
		return m
	}

	for _, item := range metadata.Items {
		switch item.Key {
		case "ANTHROPIC_MODEL":
			m.Agent = item.Value
		case "MAX_ITERATIONS":
			if parsed, err := strconv.Atoi(item.Value); err == nil {
				m.MaxIterations = parsed
			}
		case "BRANCH":
			m.Branch = item.Value
		}
	}

	return m
}

// gcpGitConfigValue reads a git config value from the host machine.
// Returns empty string if the value is not set or git is not available.
func gcpGitConfigValue(key string) string {
	out, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// isNotFoundError checks whether a GCP/gcloud error indicates a resource was not found.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "notfound") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404") ||
		strings.Contains(msg, "commandexception") ||
		strings.Contains(msg, "matched no objects or files")
}
