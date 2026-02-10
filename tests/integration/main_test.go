package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rickihastings/spinner/tests/testutil"
)

// init registers the shared resource accessors for testutil package
func init() {
	testutil.SetSharedImageAccessor(func() string {
		return sharedImageName
	})
	testutil.SetSharedBinaryAccessor(func() string {
		return sharedBinaryPath
	})
}

// Shared resources for GCP integration tests
var (
	sharedBinaryPath string
	sharedImageName  string
	gcpConfig        *testutil.GCPTestConfig
)

// TestMain provides global setup and teardown for all integration tests.
// This runs once before any tests start and once after all tests complete.
// For GCP tests, it builds the CLI once and creates a shared base image.
func TestMain(m *testing.M) {
	var cleanupSharedImage func()

	// Setup shared GCP resources if available
	gcpConfig = testutil.GCPTestConfigFromEnv()
	if gcpConfig != nil {
		fmt.Println("Setting up shared GCP test resources...")

		// Build CLI once
		sharedBinaryPath = buildSharedBinary()
		fmt.Printf("Built shared binary: %s\n", sharedBinaryPath)

		// Generate deterministic image name with config hash
		configHash := testutil.ComputeConfigHash(
			gcpConfig.Project,
			gcpConfig.Zone,
			gcpConfig.Bucket,
		)
		sharedImageName = testutil.GenerateSharedImageName(1,
			gcpConfig.Project,
			gcpConfig.Zone,
			gcpConfig.Bucket,
		)

		// Check if we should reuse existing image
		// The image name already encodes the config hash, so existence is sufficient
		// for validation — no need to also check image labels.
		forceRebuild := os.Getenv("SPINNER_TEST_FORCE_REBUILD") == "1"
		imageExists := testutil.GCPImageExists(nil, gcpConfig.Project, sharedImageName)

		var success bool

		if imageExists && !forceRebuild {
			fmt.Printf("Reusing existing GCP image: %s\n", sharedImageName)

			success = true
		} else {
			if forceRebuild {
				fmt.Println("Force rebuild enabled, recreating image...")
			} else {
				fmt.Printf("Image %s not found, creating...\n", sharedImageName)
			}

			success = createSharedGCPImage(gcpConfig, sharedImageName, configHash)
		}

		if success {
			fmt.Printf("Shared GCP image ready: %s\n", sharedImageName)

			// Keep image by default for reuse (tests are too slow to rebuild each run).
			// Set SPINNER_TEST_DELETE_IMAGE=1 to clean up after tests.
			cleanupSharedImage = func() {
				if os.Getenv("SPINNER_TEST_DELETE_IMAGE") == "1" {
					fmt.Println("Cleaning up shared GCP test resources...")
					testutil.RemoveGCPImage(nil, gcpConfig.Project, sharedImageName)

					return
				}

				fmt.Printf("Keeping shared image for reuse: %s\n", sharedImageName)
			}
		} else {
			fmt.Println("WARNING: Failed to create shared GCP image. GCP tests will be skipped.")
			// Reset to nil so tests know GCP is not available
			sharedImageName = ""
		}
	}

	// Run all tests
	code := m.Run()

	// Cleanup shared image if it was created
	if cleanupSharedImage != nil {
		cleanupSharedImage()
	}

	// Global teardown: clean up any leftover spinner data directories
	// This removes ~/.spinner/spinner-test-* directories created during tests
	_ = testutil.CleanupTestSpinnerDirs()

	os.Exit(code)
}

// buildSharedBinary builds the spinner CLI once for all tests.
// This avoids rebuilding the binary 40+ times across the test suite.
func buildSharedBinary() string {
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		fmt.Printf("Failed to get project root: %v\n", err)
		os.Exit(1)
	}

	outputPath := filepath.Join(projectRoot, "dist", "spinner")
	cmd := exec.Command("go", "build", "-o", outputPath)
	cmd.Dir = projectRoot

	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to build shared binary: %s\n", string(output))
		os.Exit(1)
	}

	return outputPath
}

// createSharedGCPImage creates a single base GCP image for all tests.
// This avoids creating 18+ separate images (5+ minutes each).
// Returns true if successful, false if failed (allowing tests to skip GCP).
func createSharedGCPImage(cfg *testutil.GCPTestConfig, imageName, configHash string) bool {
	// Delete existing image if present (we checked validity earlier)
	if testutil.GCPImageExists(nil, cfg.Project, imageName) {
		fmt.Printf("Removing existing image: %s\n", imageName)
		testutil.RemoveGCPImage(nil, cfg.Project, imageName)
	}

	// Set config hash as environment variable so it can be passed to the bake process
	// The GCP provider will read this and add it as an image label
	err := os.Setenv("SPINNER_CONFIG_HASH", configHash)
	if err != nil {
		_ = fmt.Errorf("unable to set SPRING_CONFIG_HASH: %s\n", err)
		return false
	}

	defer func() {
		_ = os.Unsetenv("SPINNER_CONFIG_HASH")
	}()

	args := []string{
		sharedBinaryPath,
		"setup",
		"--backend", "gcp",
		"--name", imageName,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}

	cmd := exec.Command(args[0], args[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to create shared GCP image: %s\n", string(output))
		return false
	}

	fmt.Printf("Shared image created successfully: %s\n", imageName)

	return true
}
