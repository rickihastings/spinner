package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

// setupTestImage is a helper that builds the CLI, creates a test image, and sets up cleanup
func setupTestImage(t *testing.T, setupArgs ...string) (imageTag string, imageName string) {
	t.Helper()

	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	imageTag = testutil.GenerateTestImageTag(t)
	imageName = "spinner:" + imageTag

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	// Run setup command with provided args plus the name flag
	args := append([]string{"setup", "--name", imageTag}, setupArgs...)
	testutil.RunCommandExpectSuccess(t, args...)

	return imageTag, imageName
}

// runContainerWithImage starts a container with the given image and returns the container name
func runContainerWithImage(t *testing.T, imageName string) string {
	t.Helper()

	containerName := testutil.GenerateTestContainerName(t)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	cmd := exec.Command("docker", "run", "-d", "--name", containerName, imageName, "tail", "-f", "/dev/null")
	err := cmd.Run()
	require.NoError(t, err, "should start container")

	return containerName
}

// execInContainer runs a command in a container and returns the output
func execInContainer(t *testing.T, containerName string, command ...string) string {
	t.Helper()

	args := append([]string{"exec", containerName}, command...)
	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	require.NoError(t, err, "command should succeed in container")

	return string(output)
}

// TestSetup_BasicBuild tests basic setup scenarios with different configurations
func TestSetup_BasicBuild(t *testing.T) {
	tests := []struct {
		name       string
		setupArgs  []string
		wantOutput string
	}{
		{
			name:       "default base image",
			setupArgs:  []string{},
			wantOutput: "Docker image built successfully",
		},
		{
			name:       "custom base image",
			setupArgs:  []string{"--base-image", "ubuntu:22.04"},
			wantOutput: "Docker image built successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.SkipIfDockerNotAvailable(t)
			testutil.BuildCLI(t)

			imageTag := testutil.GenerateTestImageTag(t)
			imageName := "spinner:" + imageTag

			t.Cleanup(func() {
				testutil.RemoveDockerImage(t, imageName)
			})

			// Run setup command
			args := append([]string{"setup", "--name", imageTag}, tt.setupArgs...)
			stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
			output := stdout + stderr

			// Verify success message
			assert.Contains(t, output, tt.wantOutput, "should show success message")

			// Verify image exists
			assert.True(t, testutil.DockerImageExists(t, imageName), "Docker image should exist after setup")
		})
	}
}

// TestSetup_InstalledTools tests that required tools are installed in the created image
func TestSetup_InstalledTools(t *testing.T) {
	tests := []struct {
		name            string
		command         []string
		wantOutputMatch string
	}{
		{
			name:            "git installed",
			command:         []string{"git", "--version"},
			wantOutputMatch: "git version",
		},
		{
			name:            "claude installed",
			command:         []string{"claude", "--version"},
			wantOutputMatch: "", // Just verify it runs successfully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build image once for this test
			_, imageName := setupTestImage(t)

			// Run container with the image
			containerName := runContainerWithImage(t, imageName)

			// Execute command in container
			output := execInContainer(t, containerName, tt.command...)

			// Verify output if needed
			if tt.wantOutputMatch != "" {
				assert.Contains(t, strings.ToLower(output), tt.wantOutputMatch, "should show expected output")
			}

			assert.NotEmpty(t, output, "command should produce output")
		})
	}
}
