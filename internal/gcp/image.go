package gcp

import (
	"context"
	"fmt"
	"time"
)

// bakeConfig holds configuration for baking a GCP image.
type bakeConfig struct {
	// ImageName is the name for the resulting custom image.
	ImageName string

	// Project is the GCP project ID.
	Project string

	// Zone is the GCP zone for the temporary bake VM.
	Zone string

	// MachineType is the machine type for the bake VM (e.g., "e2-standard-2").
	MachineType string

	// DiskSizeGB is the boot disk size in GB for the bake VM.
	DiskSizeGB int64

	// StartupScript is the bake script content (from LoadBakeScript).
	StartupScript string

	// ExtraMetadata holds additional metadata key-value pairs for the bake VM
	// (e.g., startup-script-runtime containing the startup.sh content).
	ExtraMetadata map[string]string
}

const (
	// bakeVMPrefix is the prefix for temporary bake VM names.
	bakeVMPrefix = "spinner-bake-"

	// bakeBaseImage is the base OS image for bake VMs.
	bakeBaseImage = "ubuntu-2204-lts"

	// bakeBaseImageProject is the GCP project hosting the base OS image.
	bakeBaseImageProject = "ubuntu-os-cloud"
)

// bakePollInterval and bakeTimeout are variables (not constants) so tests can override them.
var (
	// bakePollInterval is how often to check bake VM status.
	bakePollInterval = 10 * time.Second

	// bakeTimeout is the maximum time to wait for bake to complete.
	bakeTimeout = 30 * time.Minute
)

// bakeImage creates a custom GCP image by running the bake script on a temporary VM.
//
// Flow:
//  1. Create temp VM with bake script as startup-script
//  2. Wait for VM to reach TERMINATED state (script shuts down after installing tools)
//  3. Create custom image from the VM's boot disk
//  4. Delete the temporary VM
//
// The temp VM is always cleaned up, even if image creation fails.
func bakeImage(ctx context.Context, client Client, config bakeConfig) error {
	bakeVMName := bakeVMPrefix + config.ImageName

	// Step 1: Create temporary bake VM
	fmt.Printf("Creating temporary bake VM: %s\n", bakeVMName)

	metadata := map[string]string{
		"startup-script": config.StartupScript,
	}
	for k, v := range config.ExtraMetadata {
		metadata[k] = v
	}

	err := client.CreateInstance(ctx, instanceConfig{
		Name:         bakeVMName,
		Project:      config.Project,
		Zone:         config.Zone,
		MachineType:  config.MachineType,
		ImageProject: bakeBaseImageProject,
		ImageName:    bakeBaseImage,
		DiskSizeGB:   config.DiskSizeGB,
		Network:      "default",
		ExternalIP:   true,
		Metadata:     metadata,
		Labels: map[string]string{
			"spinner-managed": "true",
			"spinner-purpose": "bake",
			"spinner-image":   config.ImageName,
		},
		Scopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_only",
			"https://www.googleapis.com/auth/logging.write",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create bake VM: %w", err)
	}

	// Ensure temp VM is always cleaned up
	defer func() {
		fmt.Printf("Cleaning up bake VM: %s\n", bakeVMName)

		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if deleteErr := client.DeleteInstance(cleanupCtx, config.Project, config.Zone, bakeVMName); deleteErr != nil {
			fmt.Printf("Warning: failed to delete bake VM %s: %s\n", bakeVMName, deleteErr)
		}
	}()

	// Step 2: Wait for VM to shut down (bake script calls shutdown -h now)
	fmt.Println("Waiting for bake to complete (this may take several minutes)...")

	if err := waitForVMTerminated(ctx, client, config.Project, config.Zone, bakeVMName); err != nil {
		return fmt.Errorf("bake failed: %w", err)
	}

	fmt.Println("Bake VM has shut down, creating image...")

	// Step 3: Create custom image from the bake VM's boot disk
	sourceDisk := fmt.Sprintf(
		"projects/%s/zones/%s/disks/%s",
		config.Project, config.Zone, bakeVMName,
	)

	err = client.CreateImage(ctx, config.Project, imageConfig{
		Name:        config.ImageName,
		SourceDisk:  sourceDisk,
		Description: fmt.Sprintf("Spinner baked image: %s", config.ImageName),
		Labels: map[string]string{
			"spinner-managed": "true",
			"spinner-image":   config.ImageName,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	// Step 4: Cleanup is handled by defer above
	fmt.Printf("Image created: %s\n", config.ImageName)

	return nil
}

// waitForVMTerminated polls VM status until it reaches TERMINATED state or the context expires.
func waitForVMTerminated(ctx context.Context, client Client, project, zone, name string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, bakeTimeout)
	defer cancel()

	ticker := time.NewTicker(bakePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("bake timed out after %s", bakeTimeout)
			}

			return timeoutCtx.Err()
		case <-ticker.C:
			instance, err := client.GetInstance(ctx, project, zone, name)
			if err != nil {
				return fmt.Errorf("failed to check bake VM status: %w", err)
			}

			status := vmStatus(instance.GetStatus())

			switch status {
			case vmStatusTerminated:
				return nil
			case vmStatusRunning, vmStatusProvisioning, vmStatusStaging, vmStatusStopping:
				// Still working, continue polling
				continue
			default:
				return fmt.Errorf("bake VM entered unexpected state: %s", status)
			}
		}
	}
}
