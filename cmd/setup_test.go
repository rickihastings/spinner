package cmd

import (
	"bytes"
	"testing"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/stretchr/testify/assert"
)

// TestSetupCommand_MissingNameFlag tests that the setup command returns an error when --name is missing
func TestSetupCommand_MissingNameFlag(t *testing.T) {
	mockClient := new(docker.MockDockerClient)
	cmd := NewSetupCommand(mockClient)

	// Capture output
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required flag: --name")
	// Verify no Docker operations were attempted
	mockClient.AssertNotCalled(t, "BuildImage")
}
