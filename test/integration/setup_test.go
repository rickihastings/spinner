package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rickihastings/spinner/test/testutil"
)

// TestSetup_BasicBuild tests basic setup scenarios with different configurations
func TestDockerSetup_BasicBuild(t *testing.T) {
	tests := []struct {
		name       string
		setupArgs  []string
		wantOutput string
	}{
		{
			name:       "default base image",
			setupArgs:  []string{},
			wantOutput: "Environment provisioned",
		},
		{
			name:       "with provider-args no-cache",
			setupArgs:  []string{"--provider-args", "--no-cache"},
			wantOutput: "Environment provisioned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.SkipIfDockerNotAvailable(t)

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
func TestDockerSetup_InstalledTools(t *testing.T) {
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
			_, imageName := testutil.SetupTestImage(t)

			// Run container with the image
			containerName := testutil.RunContainerWithImage(t, imageName)

			// Execute command in container
			output := testutil.ExecInContainer(t, containerName, tt.command...)

			// Verify output if needed
			if tt.wantOutputMatch != "" {
				assert.Contains(t, strings.ToLower(output), tt.wantOutputMatch, "should show expected output")
			}

			assert.NotEmpty(t, output, "command should produce output")
		})
	}
}
