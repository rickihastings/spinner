package testutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// GCP test configuration is provided via environment variables:
//   - SPINNER_TEST_GCP_PROJECT: GCP project ID
//   - SPINNER_TEST_GCP_ZONE: GCP zone (default: us-central1-a)
//   - SPINNER_TEST_GCP_BUCKET: GCS bucket for state persistence
//   - SPINNER_TEST_SERVICE_ACCOUNT: GCP service account email (optional)
//
// The first three must be set for GCP integration tests to run.

// GCPTestConfig holds the resolved configuration for GCP integration tests.
type GCPTestConfig struct {
	Project        string
	Zone           string
	Bucket         string
	ServiceAccount string
}

// GCPTestConfigFromEnv reads GCP test configuration from environment variables.
// Returns nil if any required variable is missing.
func GCPTestConfigFromEnv() *GCPTestConfig {
	project := os.Getenv("SPINNER_TEST_GCP_PROJECT")
	zone := os.Getenv("SPINNER_TEST_GCP_ZONE")
	bucket := os.Getenv("SPINNER_TEST_GCP_BUCKET")
	serviceAccount := os.Getenv("SPINNER_TEST_SERVICE_ACCOUNT")

	if project == "" || bucket == "" {
		return nil
	}

	if zone == "" {
		zone = "us-central1-a"
	}

	return &GCPTestConfig{
		Project:        project,
		Zone:           zone,
		Bucket:         bucket,
		ServiceAccount: serviceAccount,
	}
}

// SkipIfGCPNotAvailable skips the test if GCP credentials or configuration
// are not available. It checks:
//  1. Not in short mode
//  2. Required env vars are set (SPINNER_TEST_GCP_PROJECT, SPINNER_TEST_GCP_BUCKET)
//  3. Application Default Credentials (ADC) are available
func SkipIfGCPNotAvailable(t *testing.T) *GCPTestConfig {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping GCP integration test in short mode")
	}

	cfg := GCPTestConfigFromEnv()
	if cfg == nil {
		t.Skip("skipping: GCP test environment not configured (set SPINNER_TEST_GCP_PROJECT, SPINNER_TEST_GCP_BUCKET)")
	}

	// Check that gcloud is available and authenticated
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	if err := cmd.Run(); err != nil {
		t.Skipf("skipping: GCP credentials not available (gcloud auth print-access-token failed): %v", err)
	}

	return cfg
}

// GenerateTestImageName creates a unique GCP image name for testing.
// Format: spinner-test-<timestamp> (valid GCP image name).
func GenerateTestImageName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-%d", time.Now().Unix())
}

// GCPInstanceExists checks if a GCP VM instance exists using gcloud CLI.
// Returns true if the instance exists (any state), false if not found.
func GCPInstanceExists(t *testing.T, project, zone, name string) bool {
	t.Helper()

	cmd := exec.Command("gcloud", "compute", "instances", "describe", name,
		"--project", project,
		"--zone", zone,
		"--format", "value(status)",
	)
	err := cmd.Run()

	return err == nil
}

// GCPInstanceStatus returns the status of a GCP VM instance (e.g., "RUNNING", "TERMINATED").
// Returns empty string if the instance doesn't exist.
func GCPInstanceStatus(t *testing.T, project, zone, name string) string {
	t.Helper()

	cmd := exec.Command("gcloud", "compute", "instances", "describe", name,
		"--project", project,
		"--zone", zone,
		"--format", "value(status)",
	)

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// GCPInstanceMetadata returns the value of a specific metadata key for a GCP VM instance.
// Returns the value and true if found, empty string and false if not found.
func GCPInstanceMetadata(t *testing.T, project, zone, name, key string) (string, bool) {
	t.Helper()

	cmd := exec.Command("gcloud", "compute", "instances", "describe", name,
		"--project", project,
		"--zone", zone,
		"--format", fmt.Sprintf("value(metadata.items.filter(key:%s).extract(value).flatten())", key),
	)

	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	val := strings.TrimSpace(string(output))
	if val == "" {
		return "", false
	}

	return val, true
}

// StopGCPInstance stops a running GCP VM instance.
func StopGCPInstance(t *testing.T, project, zone, name string) {
	t.Helper()

	cmd := exec.Command("gcloud", "compute", "instances", "stop", name,
		"--project", project,
		"--zone", zone,
		"--quiet",
	)

	err := cmd.Run()
	if err != nil {
		t.Logf("Warning: failed to stop GCP instance %s: %v", name, err)
	}
}

// RemoveGCPInstance deletes a GCP VM instance if it exists.
func RemoveGCPInstance(t *testing.T, project, zone, name string) {
	t.Helper()

	if !GCPInstanceExists(t, project, zone, name) {
		return
	}

	cmd := exec.Command("gcloud", "compute", "instances", "delete", name,
		"--project", project,
		"--zone", zone,
		"--quiet",
	)

	err := cmd.Run()
	if err != nil {
		t.Logf("Warning: failed to remove GCP instance %s: %v", name, err)
	}
}

// GCPImageExists checks if a GCP Compute Engine image exists.
func GCPImageExists(t *testing.T, project, name string) bool {
	if t != nil {
		t.Helper()
	}

	cmd := exec.Command("gcloud", "compute", "images", "describe", name,
		"--project", project,
		"--format", "value(name)",
	)
	err := cmd.Run()

	return err == nil
}

// RemoveGCPImage deletes a GCP Compute Engine image if it exists.
func RemoveGCPImage(t *testing.T, project, name string) {
	if t != nil {
		t.Helper()
	}

	if !GCPImageExists(t, project, name) {
		return
	}

	cmd := exec.Command("gcloud", "compute", "images", "delete", name,
		"--project", project,
		"--quiet",
	)

	err := cmd.Run()
	if err != nil && t != nil {
		t.Logf("Warning: failed to remove GCP image %s: %v", name, err)
	}
}

// ComputeConfigHash generates an 8-character hash from bake configuration.
// This hash ensures that images created with different configs get different names.
func ComputeConfigHash(project, zone, bucket string) string {
	// Concatenate all config values that affect the bake
	data := fmt.Sprintf("%s:%s:%s", project, zone, bucket)

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(data))

	// Return first 8 hex characters
	return fmt.Sprintf("%x", hash[:4])
}

// GenerateSharedImageName creates a deterministic image name based on version and config.
// Format: test-shared-v{version}-{hash}
// Version allows manual invalidation on major changes, hash ensures config match.
func GenerateSharedImageName(version int, project, zone, bucket string) string {
	hash := ComputeConfigHash(project, zone, bucket)
	return fmt.Sprintf("test-shared-v%d-%s", version, hash)
}

// GetImageLabels retrieves labels from a GCP Compute Engine image using gcloud CLI.
// Returns empty map if image doesn't exist or has no labels.
func GetImageLabels(t *testing.T, project, imageName string) map[string]string {
	if t != nil {
		t.Helper()
	}

	cmd := exec.Command("gcloud", "compute", "images", "describe", imageName,
		"--project", project,
		"--format", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return map[string]string{}
	}

	var result struct {
		Labels map[string]string `json:"labels"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		if t != nil {
			t.Logf("Warning: failed to parse image labels for %s: %v", imageName, err)
		}

		return map[string]string{}
	}

	if result.Labels == nil {
		return map[string]string{}
	}

	return result.Labels
}

// ValidateSharedImage checks if an existing image matches the expected configuration.
// Returns true if the image exists AND has a matching spinner-config-hash label.
func ValidateSharedImage(t *testing.T, project, imageName, expectedHash string) bool {
	if t != nil {
		t.Helper()
	}

	// Check if image exists
	if !GCPImageExists(t, project, imageName) {
		return false
	}

	// Get image labels
	labels := GetImageLabels(t, project, imageName)

	// Verify config hash matches
	actualHash, ok := labels["spinner-config-hash"]
	if !ok {
		if t != nil {
			t.Logf("Image %s exists but has no spinner-config-hash label", imageName)
		}

		return false
	}

	if actualHash != expectedHash {
		if t != nil {
			t.Logf("Image %s has mismatched config hash: expected %s, got %s", imageName, expectedHash, actualHash)
		}

		return false
	}

	return true
}

// WriteGCSObject writes data to a GCS object (for test setup, e.g., mock log files).
func WriteGCSObject(t *testing.T, bucket, object string, data []byte) {
	t.Helper()

	cmd := exec.Command("gcloud", "storage", "cp", "-", fmt.Sprintf("gs://%s/%s", bucket, object))
	cmd.Stdin = bytes.NewReader(data)

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to write GCS object %s/%s: %v\n%s", bucket, object, err, output)
	}
}

// GCSObjectExists checks whether a GCS object exists.
func GCSObjectExists(t *testing.T, bucket, object string) bool {
	t.Helper()

	cmd := exec.Command("gcloud", "storage", "ls", fmt.Sprintf("gs://%s/%s", bucket, object))
	err := cmd.Run()

	return err == nil
}

// ReadGCSObject reads the full contents of a GCS object.
func ReadGCSObject(t *testing.T, bucket, object string) []byte {
	t.Helper()

	cmd := exec.Command("gcloud", "storage", "cat", fmt.Sprintf("gs://%s/%s", bucket, object))

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to read GCS object %s/%s: %v", bucket, object, err)
	}

	return output
}

// CleanupGCSPrefix removes all objects under a given prefix in a GCS bucket.
// Used to clean up state/log objects created during tests.
func CleanupGCSPrefix(t *testing.T, bucket, prefix string) {
	t.Helper()

	cmd := exec.Command("gcloud", "storage", "rm", fmt.Sprintf("gs://%s/%s**", bucket, prefix), "--recursive")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore "matched no objects or files" — prefix may be empty or never created
		if !strings.Contains(string(output), "matched no objects or files") {
			t.Logf("Warning: failed to cleanup GCS prefix %s/%s: %v\n%s", bucket, prefix, err, output)
		}
	}
}

// GetSharedGCPImage returns the shared test image name created in TestMain.
// This is the preferred method for GCP integration tests to avoid recreating
// images (which takes 35+ minutes each).
//
// The shared image is created once before all tests run and deleted after all tests complete.
// Individual tests create separate VM instances from this shared image, ensuring test isolation.
func GetSharedGCPImage(t *testing.T) string {
	t.Helper()

	// Access the shared image name from the integration package
	// This is set by TestMain in tests/integration/main_test.go
	imageName := getSharedImageName()
	if imageName == "" {
		t.Skip("Shared GCP image not available (GCP test environment not configured)")
	}

	return imageName
}

// getSharedImageName is implemented in integration package to access the global variable.
// This is a workaround for accessing package-level variables across packages.
var getSharedImageName = func() string {
	return ""
}

// SetSharedImageAccessor allows the integration package to register its image accessor.
// This is called from init() in the integration package.
func SetSharedImageAccessor(accessor func() string) {
	getSharedImageName = accessor
}

// RunGCPSpinCommand executes the spin command with GCP backend and returns the instance name from output.
func RunGCPSpinCommand(t *testing.T, args ...string) (instanceName string, stdout string, stderr string) {
	t.Helper()

	stdout, stderr = RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Extract instance name from output
	// Expected format: "Instance created successfully: <instance-name>"
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Instance created successfully:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				instanceName = parts[len(parts)-1]
				break
			}
		}

		if strings.Contains(line, "Reusing running instance:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				instanceName = parts[len(parts)-1]
				break
			}
		}

		if strings.Contains(line, "Instance restarted:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				instanceName = parts[len(parts)-1]
				break
			}
		}
	}

	return instanceName, stdout, stderr
}

// WaitForGCSStateStatus polls GCS state.json until it reaches the expected status or timeout.
func WaitForGCSStateStatus(t *testing.T, bucket, instanceName, expectedStatus string, timeoutSec int) {
	t.Helper()

	objectPath := fmt.Sprintf("%s/state.json", instanceName)
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)

	for time.Now().Before(deadline) {
		if GCSObjectExists(t, bucket, objectPath) {
			data := ReadGCSObject(t, bucket, objectPath)

			var state struct {
				Status string `json:"status"`
			}

			if err := json.Unmarshal(data, &state); err == nil {
				if state.Status == expectedStatus {
					return
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf("Timeout waiting for GCS state status %s after %d seconds", expectedStatus, timeoutSec)
}

// WaitForGCPInstanceStatus polls VM status until it reaches the expected status or timeout.
func WaitForGCPInstanceStatus(t *testing.T, project, zone, instanceName, expectedStatus string, timeoutSec int) {
	t.Helper()

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)

	for time.Now().Before(deadline) {
		status := GCPInstanceStatus(t, project, zone, instanceName)

		if status == expectedStatus {
			return
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf("Timeout waiting for VM status %s after %d seconds", expectedStatus, timeoutSec)
}

// Sleep pauses execution for the specified duration (convenience wrapper for tests).
func Sleep(t *testing.T, seconds int) {
	t.Helper()
	t.Logf("Sleeping for %d seconds...", seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
}
