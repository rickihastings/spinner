package cmd

import (
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// testFactory wraps a MockProvider into a Factory with the "docker" backend.
func testFactory(mockProv *provider.MockProvider) *provider.Factory {
	f := provider.NewFactory()

	f.Register(provider.BackendDocker, func() (provider.Provider, error) {
		return mockProv, nil
	})

	return f
}

// testGCPFactory creates a Factory with both docker and gcp backends using mocks.
func testGCPFactory(mockProv *provider.MockProvider) *provider.Factory {
	f := provider.NewFactory()

	f.Register(provider.BackendDocker, func() (provider.Provider, error) {
		return mockProv, nil
	})

	f.Register(provider.BackendGCP, func() (provider.Provider, error) {
		return mockProv, nil
	})

	return f
}

// setupSpinCommandWithMocks creates a spin command with common mock expectations
// for tests that need a working spin command (valid env, instance creation).
func setupSpinCommandWithMocks(t *testing.T) *cobra.Command {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	mockProvider := new(provider.MockProvider)
	mockProvider.On("InstanceName", mock.Anything).Return("test-container")
	mockProvider.On("Status", mock.Anything, "test-container").Return(provider.InstanceStatusNone, nil)
	mockProvider.On("Create", mock.Anything, mock.Anything).Return(
		&provider.Instance{Name: "test-container", Status: provider.InstanceStatusRunning}, nil,
	)

	return NewSpinCommand(testFactory(mockProvider))
}

// TestMergeProviderArgs_CLIOnly tests mergeProviderArgs with only CLI args
func TestMergeProviderArgs_CLIOnly(t *testing.T) {
	result := mergeProviderArgs([]string{"-v /data:/data", "--network=host"})
	assert.Equal(t, []string{"-v /data:/data", "--network=host"}, result)
}

// TestMergeProviderArgs_Empty tests mergeProviderArgs with no args from any source
func TestMergeProviderArgs_Empty(t *testing.T) {
	result := mergeProviderArgs(nil)
	assert.Empty(t, result)
}

// TestMergeProviderArgs_EmptySlice tests mergeProviderArgs with empty slice
func TestMergeProviderArgs_EmptySlice(t *testing.T) {
	result := mergeProviderArgs([]string{})
	assert.Empty(t, result)
}
