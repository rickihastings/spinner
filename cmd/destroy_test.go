package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestDestroyCommand_SingleRunningInstance tests destroying a single running instance
func TestDestroyCommand_SingleRunningInstance(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	mockProvider.On("Status", mock.Anything, "test-instance").Return(provider.InstanceStatusRunning, nil)
	mockProvider.On("Remove", mock.Anything, "test-instance").Return(nil)

	cmd := NewDestroyCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"test-instance"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertCalled(t, "Status", mock.Anything, "test-instance")
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "test-instance")
	assert.Contains(t, b.String(), "✓ Instance 'test-instance' destroyed")
}

// TestDestroyCommand_SingleStoppedInstance tests destroying a single stopped instance
func TestDestroyCommand_SingleStoppedInstance(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	mockProvider.On("Status", mock.Anything, "stopped-instance").Return(provider.InstanceStatusStopped, nil)
	mockProvider.On("Remove", mock.Anything, "stopped-instance").Return(nil)

	cmd := NewDestroyCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"stopped-instance"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertCalled(t, "Status", mock.Anything, "stopped-instance")
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "stopped-instance")
	assert.Contains(t, b.String(), "✓ Instance 'stopped-instance' destroyed")
}

// TestDestroyCommand_MultipleInstances tests destroying multiple instances
func TestDestroyCommand_MultipleInstances(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	mockProvider.On("Status", mock.Anything, "instance1").Return(provider.InstanceStatusRunning, nil)
	mockProvider.On("Status", mock.Anything, "instance2").Return(provider.InstanceStatusStopped, nil)
	mockProvider.On("Status", mock.Anything, "instance3").Return(provider.InstanceStatusRunning, nil)
	mockProvider.On("Remove", mock.Anything, "instance1").Return(nil)
	mockProvider.On("Remove", mock.Anything, "instance2").Return(nil)
	mockProvider.On("Remove", mock.Anything, "instance3").Return(nil)

	cmd := NewDestroyCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"instance1", "instance2", "instance3"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "instance1")
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "instance2")
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "instance3")
	assert.Contains(t, b.String(), "✓ Instance 'instance1' destroyed")
	assert.Contains(t, b.String(), "✓ Instance 'instance2' destroyed")
	assert.Contains(t, b.String(), "✓ Instance 'instance3' destroyed")
}

// TestDestroyCommand_InstanceNotFound tests error when instance doesn't exist
func TestDestroyCommand_InstanceNotFound(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	mockProvider.On("Status", mock.Anything, "nonexistent").Return(provider.InstanceStatusNone, nil)

	cmd := NewDestroyCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to destroy 1 of 1 instance(s)")
	assert.Contains(t, b.String(), "✗ Instance 'nonexistent' not found")
	mockProvider.AssertNotCalled(t, "Remove", mock.Anything, "nonexistent")
}

// TestDestroyCommand_PartialFailure tests partial failure with multiple instances
func TestDestroyCommand_PartialFailure(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	mockProvider.On("Status", mock.Anything, "instance1").Return(provider.InstanceStatusRunning, nil)
	mockProvider.On("Status", mock.Anything, "instance2").Return(provider.InstanceStatusNone, nil)
	mockProvider.On("Status", mock.Anything, "instance3").Return(provider.InstanceStatusRunning, nil)
	mockProvider.On("Remove", mock.Anything, "instance1").Return(nil)
	mockProvider.On("Remove", mock.Anything, "instance3").Return(nil)

	cmd := NewDestroyCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"instance1", "instance2", "instance3"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to destroy 1 of 3 instance(s)")
	assert.Contains(t, b.String(), "✓ Instance 'instance1' destroyed")
	assert.Contains(t, b.String(), "✗ Instance 'instance2' not found")
	assert.Contains(t, b.String(), "✓ Instance 'instance3' destroyed")
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "instance1")
	mockProvider.AssertNotCalled(t, "Remove", mock.Anything, "instance2")
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "instance3")
}

// TestDestroyCommand_MissingArgument tests error when no instance name is provided
func TestDestroyCommand_MissingArgument(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewDestroyCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 arg(s)")
}

// TestDestroyCommand_StateDirectoryCleanup tests that state directory is removed
func TestDestroyCommand_StateDirectoryCleanup(t *testing.T) {
	// Create a temporary home directory
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		os.Setenv("HOME", originalHome)
	}()

	// Create state directory
	instanceName := "cleanup-test-instance"
	stateDir := filepath.Join(tmpHome, ".spinner", instanceName)
	logsDir := filepath.Join(stateDir, "logs")
	stateSubDir := filepath.Join(stateDir, "state")

	err := os.MkdirAll(logsDir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(stateSubDir, 0755)
	assert.NoError(t, err)

	// Create some files
	err = os.WriteFile(filepath.Join(logsDir, "test.log"), []byte("test log"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(stateSubDir, "state.json"), []byte("{}"), 0644)
	assert.NoError(t, err)

	// Verify directory exists
	_, err = os.Stat(stateDir)
	assert.NoError(t, err)

	// Mock provider - simulate state cleanup like real providers do
	mockProvider := new(provider.MockProvider)
	mockProvider.On("Status", mock.Anything, instanceName).Return(provider.InstanceStatusRunning, nil)
	mockProvider.On("Remove", mock.Anything, instanceName).Run(func(args mock.Arguments) {
		// Simulate provider removing state directory
		_ = os.RemoveAll(stateDir)
	}).Return(nil)

	cmd := NewDestroyCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{instanceName})

	err = cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, b.String(), "✓ Instance 'cleanup-test-instance' destroyed")

	// Verify state directory is removed
	_, err = os.Stat(stateDir)
	assert.True(t, os.IsNotExist(err), "State directory should be removed")
}
