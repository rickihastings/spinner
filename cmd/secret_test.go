package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rickihastings/spinner/internal/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupBlobWithKeyFile creates an encrypted blob and key file in a temp directory.
// It overrides defaultBlobPath and defaultKeyPath for the test.
// Callers must defer the returned cleanup function.
func setupBlobWithKeyFile(t *testing.T, secrets map[string]string) func() {
	t.Helper()

	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")
	keyFilePath := filepath.Join(dir, "secrets.key")

	key, blob, err := secret.EncryptBlobWithKey(secrets)
	if err != nil {
		t.Fatalf("EncryptBlobWithKey: %v", err)
	}

	if err := os.WriteFile(blobPath, blob, 0600); err != nil {
		t.Fatalf("WriteFile blob: %v", err)
	}

	if err := os.WriteFile(keyFilePath, key, 0600); err != nil {
		t.Fatalf("WriteFile key: %v", err)
	}

	origBlobPath := defaultBlobPath
	origKeyPath := defaultKeyPath
	defaultBlobPath = blobPath
	defaultKeyPath = keyFilePath

	return func() {
		defaultBlobPath = origBlobPath
		defaultKeyPath = origKeyPath
	}
}

// testStoreFactory returns a storeFactory that always returns the given mock.
func testStoreFactory(m *secret.MockStore) storeFactory {
	return func() secret.Store {
		return m
	}
}

func TestSecretSetCommand_WithValueFlag(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Set", "MY_TOKEN", "supersecret").Return(nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "MY_TOKEN", "--value", "supersecret"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `Secret "MY_TOKEN" stored`)
	mockStore.AssertExpectations(t)
}

func TestSecretSetCommand_EmptyValue(t *testing.T) {
	mockStore := new(secret.MockStore)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "MY_TOKEN", "--value", ""})

	// Provide empty input for the prompt path too
	// Since --value is "" the command will try to prompt; mock readPassword
	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte(""), nil
	}

	defer func() { readPassword = origReadPassword }()

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret value must not be empty")
	mockStore.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)
}

func TestSecretSetCommand_WithPromptedValue(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Set", "API_KEY", "prompted-value").Return(nil)

	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte("prompted-value"), nil
	}

	defer func() { readPassword = origReadPassword }()

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "API_KEY"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `Secret "API_KEY" stored`)
	mockStore.AssertExpectations(t)
}

func TestSecretSetCommand_StoreError(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Set", "MY_TOKEN", "val").Return(fmt.Errorf("disk full"))

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "MY_TOKEN", "--value", "val"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestSecretSetCommand_MissingKeyArg(t *testing.T) {
	mockStore := new(secret.MockStore)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestSecretListCommand_WithKeys(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("List").Return([]string{"API_KEY", "GITHUB_TOKEN"}, nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "API_KEY")
	assert.Contains(t, buf.String(), "GITHUB_TOKEN")
	mockStore.AssertExpectations(t)
}

func TestSecretListCommand_Empty(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("List").Return([]string{}, nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No secrets stored")
	mockStore.AssertExpectations(t)
}

func TestSecretListCommand_StoreError(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("List").Return(nil, fmt.Errorf("corrupt store"))

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "corrupt store")
}

func TestSecretDeleteCommand_Success(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Delete", "OLD_TOKEN").Return(nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete", "OLD_TOKEN"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `Secret "OLD_TOKEN" deleted`)
	mockStore.AssertExpectations(t)
}

func TestSecretDeleteCommand_NotFound(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Delete", "NONEXISTENT").Return(fmt.Errorf("%w: NONEXISTENT", secret.ErrNotFound))

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete", "NONEXISTENT"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestSecretDeleteCommand_MissingKeyArg(t *testing.T) {
	mockStore := new(secret.MockStore)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestSecretInjectCommand_DecryptsAndRunsCommand(t *testing.T) {
	cleanup := setupBlobWithKeyFile(t, map[string]string{
		"GITHUB_TOKEN": "ghp_test123",
		"CUSTOM_KEY":   "custom_val",
	})
	defer cleanup()

	cmd := newSecretCommand(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	// Use sh -c 'echo $GITHUB_TOKEN' to verify env injection
	cmd.SetArgs([]string{"inject", "--", "sh", "-c", "echo $GITHUB_TOKEN"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

func TestSecretInjectCommand_KeyFromEnvVar(t *testing.T) {
	// Create blob + key, set SPINNER_SECRET_KEY to key path
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")
	keyFilePath := filepath.Join(dir, "secrets.key")

	key, blob, err := secret.EncryptBlobWithKey(map[string]string{
		"MY_SECRET": "secret_value",
	})
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(blobPath, blob, 0600))
	assert.NoError(t, os.WriteFile(keyFilePath, key, 0600))

	origBlobPath := defaultBlobPath
	origKeyPath := defaultKeyPath
	defaultBlobPath = blobPath
	// Set defaultKeyPath to nonexistent so env var is used
	defaultKeyPath = filepath.Join(dir, "nonexistent.key")

	defer func() {
		defaultBlobPath = origBlobPath
		defaultKeyPath = origKeyPath
	}()

	t.Setenv("SPINNER_SECRET_KEY", keyFilePath)

	cmd := newSecretCommand(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inject", "--", "sh", "-c", "printenv MY_SECRET"})

	err = cmd.Execute()

	assert.NoError(t, err)
}

func TestSecretInjectCommand_WrongKey(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")
	wrongKeyPath := filepath.Join(dir, "wrong.key")

	_, blob, err := secret.EncryptBlobWithKey(map[string]string{"KEY": "val"})
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(blobPath, blob, 0600))

	// Write a different 32-byte key
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = 0xFF
	}

	assert.NoError(t, os.WriteFile(wrongKeyPath, wrongKey, 0600))

	origBlobPath := defaultBlobPath
	origKeyPath := defaultKeyPath
	defaultBlobPath = blobPath
	defaultKeyPath = wrongKeyPath

	defer func() {
		defaultBlobPath = origBlobPath
		defaultKeyPath = origKeyPath
	}()

	// Clear passphrase env so it doesn't fall back to passphrase
	t.Setenv("SPINNER_SECRET_PASSPHRASE", "")

	// Mock readPassword to return empty (so fallback also fails)
	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte(""), nil
	}

	defer func() { readPassword = origReadPassword }()

	cmd := newSecretCommand(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inject", "--", "echo", "should-not-run"})

	err = cmd.Execute()

	assert.Error(t, err)
}

func TestSecretInjectCommand_MissingBlob(t *testing.T) {
	dir := t.TempDir()

	origBlobPath := defaultBlobPath
	origKeyPath := defaultKeyPath
	defaultBlobPath = "/nonexistent/path/secrets.enc"
	defaultKeyPath = filepath.Join(dir, "nonexistent.key")

	defer func() {
		defaultBlobPath = origBlobPath
		defaultKeyPath = origKeyPath
	}()

	// Clear passphrase env so fallback path also fails
	t.Setenv("SPINNER_SECRET_PASSPHRASE", "")

	// Mock readPassword to return empty (so passphrase fallback also fails)
	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte(""), nil
	}

	defer func() { readPassword = origReadPassword }()

	cmd := newSecretCommand(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inject", "--", "echo", "should-not-run"})

	err := cmd.Execute()

	assert.Error(t, err)
}

func TestSecretInjectCommand_MissingCommandArg(t *testing.T) {
	t.Setenv("SPINNER_SECRET_PASSPHRASE", "test-pass")

	cmd := newSecretCommand(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inject", "--"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing command argument")
}

func TestSecretInjectCommand_PassphraseFallback(t *testing.T) {
	// When key file is missing, inject should fall back to passphrase-based decryption
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")

	blob, err := secret.EncryptBlob(map[string]string{"KEY": "val"}, "prompted-pass")
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(blobPath, blob, 0600))

	origBlobPath := defaultBlobPath
	origKeyPath := defaultKeyPath
	defaultBlobPath = blobPath
	// Set key path to nonexistent so it falls back to passphrase
	defaultKeyPath = filepath.Join(dir, "nonexistent.key")

	defer func() {
		defaultBlobPath = origBlobPath
		defaultKeyPath = origKeyPath
	}()

	// Ensure env var is NOT set so it falls back to prompt
	t.Setenv("SPINNER_SECRET_PASSPHRASE", "")

	// Mock the readPassword function to return our passphrase
	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte("prompted-pass"), nil
	}

	defer func() { readPassword = origReadPassword }()

	cmd := newSecretCommand(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inject", "--", "sh", "-c", "printenv KEY"})

	err = cmd.Execute()

	assert.NoError(t, err)
}
