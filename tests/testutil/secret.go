package testutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rickihastings/spinner/internal/secret"
)

const (
	testStorePassphrase  = "spinner-integration-test-passphrase"
	testGitHubToken      = "test-github-token"
	testClaudeOAuthToken = "test-claude-oauth-token"
)

// SetupTestSecretStore creates a temporary encrypted secret store populated with
// test tokens required by `spinner spin` (GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN).
// It also writes an empty .spinner.json config file and sets SPINNER_CONFIG_FILE so
// that test subprocesses do not inherit the developer's ~/.spinner.json defaults
// (e.g. model=...) that would cause test assertions to fail.
//
// Sets the following in the test process environment (inherited by subprocesses):
//   - SPINNER_SECRET_STORE      — path to the temp store file
//   - SPINNER_SECRET_PASSPHRASE — passphrase for the temp store
//   - SPINNER_CONFIG_FILE       — path to an empty .spinner.json (overrides config search)
//
// Returns a cleanup function that removes temp files and restores env vars.
func SetupTestSecretStore() (cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "spinner-test-store-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	// Create the secret store in the temp directory.
	// EncryptedFileStore.load() returns an empty map when the file is absent,
	// allowing Set() to create a fresh store from scratch.
	storePath := filepath.Join(tmpDir, "secrets.enc")

	store := secret.NewEncryptedFileStore(storePath, func() (string, error) {
		return testStorePassphrase, nil
	})

	if err := store.Set("GITHUB_TOKEN", testGitHubToken); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("seeding GITHUB_TOKEN: %w", err)
	}

	if err := store.Set("CLAUDE_CODE_OAUTH_TOKEN", testClaudeOAuthToken); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("seeding CLAUDE_CODE_OAUTH_TOKEN: %w", err)
	}

	// Write an empty JSON config file so SPINNER_CONFIG_FILE points to a valid
	// (but empty) config, preventing findConfigFile from traversing up to the
	// developer's home directory and picking up ~/.spinner.json defaults.
	configPath := filepath.Join(tmpDir, ".spinner.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("creating config file: %w", err)
	}

	if err := os.Setenv("SPINNER_SECRET_STORE", storePath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("setting SPINNER_SECRET_STORE: %w", err)
	}

	if err := os.Setenv("SPINNER_SECRET_PASSPHRASE", testStorePassphrase); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("setting SPINNER_SECRET_PASSPHRASE: %w", err)
	}

	if err := os.Setenv("SPINNER_CONFIG_FILE", configPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("setting SPINNER_CONFIG_FILE: %w", err)
	}

	return func() {
		_ = os.RemoveAll(tmpDir)
		_ = os.Unsetenv("SPINNER_SECRET_STORE")
		_ = os.Unsetenv("SPINNER_SECRET_PASSPHRASE")
		_ = os.Unsetenv("SPINNER_CONFIG_FILE")
	}, nil
}
