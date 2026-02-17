package cmd

import (
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rickihastings/spinner/internal/secret"
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

// installMockSecretStore installs a mock secret store that returns test tokens
// and sets a test passphrase. It restores the original store factory on cleanup.
func installMockSecretStore(t *testing.T) {
	t.Helper()

	// Set passphrase for blob encryption
	t.Setenv("SPINNER_SECRET_PASSPHRASE", "test-passphrase")

	mockStore := new(secret.MockStore)
	mockStore.On("Get", "GITHUB_TOKEN").Return("test-token", nil)
	mockStore.On("Get", "CLAUDE_CODE_OAUTH_TOKEN").Return("test-token", nil)
	mockStore.On("Get", mock.Anything).Return("test-value", nil)

	origFactory := spinStoreFactory
	spinStoreFactory = func() secret.Store { return mockStore }

	t.Cleanup(func() {
		spinStoreFactory = origFactory
	})
}

// setupSpinCommandWithMocks creates a spin command with common mock expectations
// for tests that need a working spin command (valid env, instance creation).
func setupSpinCommandWithMocks(t *testing.T) *cobra.Command {
	installMockSecretStore(t)

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
	result := mergeProviderArgs(configSpinProviderArgs, []string{"-v /data:/data", "--network=host"})
	assert.Equal(t, []string{"-v /data:/data", "--network=host"}, result)
}

// TestMergeProviderArgs_Empty tests mergeProviderArgs with no args from any source
func TestMergeProviderArgs_Empty(t *testing.T) {
	result := mergeProviderArgs(configSpinProviderArgs, nil)
	assert.Empty(t, result)
}

// TestMergeProviderArgs_EmptySlice tests mergeProviderArgs with empty slice
func TestMergeProviderArgs_EmptySlice(t *testing.T) {
	result := mergeProviderArgs(configSpinProviderArgs, []string{})
	assert.Empty(t, result)
}
