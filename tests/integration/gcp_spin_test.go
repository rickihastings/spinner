package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestGCPSpin_NonExistentImage tests that referencing a non-existent image produces a clear error.
func TestGCPSpin_NonExistentImage(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", "nonexistent-image-12345",
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.NotEqual(t, 0, exitCode, "should fail with non-existent image")
	assert.Contains(t, output, "not found", "should indicate image was not found")
}
