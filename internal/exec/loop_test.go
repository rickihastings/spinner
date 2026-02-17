package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/secret"
)

// mockExecutor is a test helper that implements agent.Executor interface
type mockExecutor struct {
	result *agent.Result
	err    error
}

func (m *mockExecutor) Execute(_ context.Context, _ string) (<-chan agent.Event, error) {
	// Not used in loop tests
	return nil, nil
}

func (m *mockExecutor) ExecuteAndCollect(_ context.Context, _ string) (*agent.Result, error) {
	return m.result, m.err
}

func TestNewRunner(t *testing.T) {
	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 5,
		Branch:        "main",
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	statePath := "/tmp/state.json"

	runner := NewRunner(config, state, statePath)

	if runner.config != config {
		t.Error("Config not set correctly")
	}

	if runner.state != state {
		t.Error("State not set correctly")
	}

	if runner.statePath != statePath {
		t.Error("State path not set correctly")
	}
}

func TestRunner_Run_MaxIterations(t *testing.T) {
	// Create temp directory for state
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 2,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor factory to return no completion
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		return &mockExecutor{
			result: &agent.Result{
				Completed:   false,
				RateLimited: false,
				AuthError:   false,
			},
			err: nil,
		}
	}

	// Mock PushChanges to always succeed
	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	// Should exit with code 1 when max iterations reached
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// State should reflect max iterations error
	if state.Status != statusError {
		t.Errorf("Expected status error, got %s", state.Status)
	}

	if state.ErrorMessage != "max iterations reached" {
		t.Errorf("Expected error message 'max iterations reached', got %s", state.ErrorMessage)
	}

	// State should have been saved
	loadedState, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if loadedState.Status != statusError {
		t.Errorf("Expected saved status error, got %s", loadedState.Status)
	}
}

func TestRunner_Run_Completion(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 5,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor factory to return completion on first iteration
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		return &mockExecutor{
			result: &agent.Result{
				Completed: true,
			},
			err: nil,
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	if state.Status != statusCompleted {
		t.Errorf("Expected status completed, got %s", state.Status)
	}

	if state.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestRunner_Run_AuthError(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 5,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		return &mockExecutor{
			result: &agent.Result{
				AuthError:    true,
				ErrorMessage: "authentication failed",
			},
			err: nil,
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	if state.Status != statusAuthError {
		t.Errorf("Expected status auth_error, got %s", state.Status)
	}

	if state.ErrorMessage != "authentication failed" {
		t.Errorf("Expected error message 'authentication failed', got %s", state.ErrorMessage)
	}
}

func TestRunner_Run_RateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limit test in short mode")
	}

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 5,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor factory to return rate limit on first call, then completion
	callCount := 0
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		callCount++
		if callCount == 1 {
			return &mockExecutor{
				result: &agent.Result{
					RateLimited:  true,
					ErrorMessage: "rate limited",
				},
				err: nil,
			}
		}

		return &mockExecutor{
			result: &agent.Result{
				Completed: true,
			},
			err: nil,
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	// Use a context with timeout to avoid waiting full 61 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exitCode := runner.Run(ctx)

	// Should get interrupted by context timeout during wait
	if exitCode != 130 {
		t.Errorf("Expected exit code 130 (interrupted), got %d", exitCode)
	}
}

func TestRunner_Run_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 5,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		return &mockExecutor{
			result: &agent.Result{},
			err:    nil,
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	exitCode := runner.Run(ctx)

	if exitCode != 130 {
		t.Errorf("Expected exit code 130, got %d", exitCode)
	}

	if state.Status != statusError {
		t.Errorf("Expected status error, got %s", state.Status)
	}

	if state.ErrorMessage != "interrupted by user" {
		t.Errorf("Expected error message 'interrupted by user', got %s", state.ErrorMessage)
	}
}

func TestRunner_Run_ConsecutiveErrors(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 100,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor factory to always return an error
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	callCount := 0

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		callCount++

		return &mockExecutor{
			result: &agent.Result{
				Error:        fmt.Errorf("claude CLI failed"),
				ErrorMessage: "exit status 1",
			},
			err: nil,
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	// Should bail out after maxConsecutiveErrors (3), not run all 100 iterations
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	if callCount != maxConsecutiveErrors {
		t.Errorf("Expected %d calls (consecutive error limit), got %d", maxConsecutiveErrors, callCount)
	}

	if state.Status != statusError {
		t.Errorf("Expected status error, got %s", state.Status)
	}

	if !strings.Contains(state.ErrorMessage, "consecutive errors") {
		t.Errorf("Expected error message to mention consecutive errors, got %s", state.ErrorMessage)
	}
}

func TestRunner_Run_ConsecutiveErrorsReset(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 10,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{
		Status:    statusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor: alternate between error and success
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	callCount := 0

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		callCount++
		// Alternate: error, error, success, error, error, success, ...
		if callCount%3 != 0 {
			return &mockExecutor{
				result: &agent.Result{
					Error:        fmt.Errorf("transient error"),
					ErrorMessage: "transient failure",
				},
				err: nil,
			}
		}

		return &mockExecutor{
			result: &agent.Result{},
			err:    nil,
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	// Should hit max iterations since errors never reach 3 consecutive
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 (max iterations), got %d", exitCode)
	}

	if state.ErrorMessage != "max iterations reached" {
		t.Errorf("Expected 'max iterations reached', got %s", state.ErrorMessage)
	}

	// Should have run all 10 iterations
	if callCount != 10 {
		t.Errorf("Expected 10 calls (all iterations), got %d", callCount)
	}
}

// createTestBlob creates an encrypted secrets blob in a temp directory and returns
// the blob path and passphrase. It also sets the secretsBlobPath package var to
// point to this temp file for the duration of the test.
func createTestBlob(t *testing.T, secrets map[string]string, passphrase string) string {
	t.Helper()

	blob, err := secret.EncryptBlob(secrets, passphrase)
	if err != nil {
		t.Fatalf("Failed to encrypt blob: %v", err)
	}

	blobPath := filepath.Join(t.TempDir(), "secrets.enc")
	if err := os.WriteFile(blobPath, blob, 0600); err != nil {
		t.Fatalf("Failed to write blob: %v", err)
	}

	return blobPath
}

func TestRunner_Run_SecretsDecryptedAndInjected(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create encrypted blob
	testSecrets := map[string]string{
		"GITHUB_TOKEN":            "gh-token-123",
		"CLAUDE_CODE_OAUTH_TOKEN": "claude-token-456",
	}
	passphrase := "test-passphrase"
	blobPath := createTestBlob(t, testSecrets, passphrase)

	// Override blob path and decryptBlobFunc
	oldBlobPath := secretsBlobPath

	defer func() { secretsBlobPath = oldBlobPath }()

	secretsBlobPath = blobPath

	oldDecryptBlob := decryptBlobFunc

	defer func() { decryptBlobFunc = oldDecryptBlob }()

	// Override osUnsetenv to track calls
	var unsetKeys []string

	oldOsUnsetenv := osUnsetenv

	defer func() { osUnsetenv = oldOsUnsetenv }()

	osUnsetenv = func(key string) error {
		unsetKeys = append(unsetKeys, key)
		return nil
	}

	// Set passphrase in env
	t.Setenv("SPINNER_SECRET_PASSPHRASE", passphrase)

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 1,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{}

	runner := NewRunner(config, state, statePath)

	var capturedEnv []string

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		capturedEnv = env

		return &mockExecutor{
			result: &agent.Result{Completed: true},
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error { return nil }

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Check that secrets were injected into env
	envMap := make(map[string]string)

	for _, e := range capturedEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["GITHUB_TOKEN"] != "gh-token-123" {
		t.Errorf("Expected GITHUB_TOKEN=gh-token-123, got %q", envMap["GITHUB_TOKEN"])
	}

	if envMap["CLAUDE_CODE_OAUTH_TOKEN"] != "claude-token-456" {
		t.Errorf("Expected CLAUDE_CODE_OAUTH_TOKEN=claude-token-456, got %q", envMap["CLAUDE_CODE_OAUTH_TOKEN"])
	}

	// Check passphrase forwarded for inception
	if envMap["SPINNER_SECRET_PASSPHRASE"] != passphrase {
		t.Errorf("Expected SPINNER_SECRET_PASSPHRASE=%s, got %q", passphrase, envMap["SPINNER_SECRET_PASSPHRASE"])
	}

	// Check SPINNER_SECRET_STORE set for inception
	if envMap["SPINNER_SECRET_STORE"] != secretsBlobPath {
		t.Errorf("Expected SPINNER_SECRET_STORE=%s, got %q", secretsBlobPath, envMap["SPINNER_SECRET_STORE"])
	}

	// Check passphrase was unset from own env
	if len(unsetKeys) != 1 || unsetKeys[0] != "SPINNER_SECRET_PASSPHRASE" {
		t.Errorf("Expected SPINNER_SECRET_PASSPHRASE to be unset, got %v", unsetKeys)
	}
}

func TestRunner_Run_MissingBlobContinuesNormally(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Set passphrase but don't create a blob file
	t.Setenv("SPINNER_SECRET_PASSPHRASE", "some-passphrase")

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 1,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{}
	runner := NewRunner(config, state, statePath)

	var capturedEnv []string

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		capturedEnv = env

		return &mockExecutor{
			result: &agent.Result{Completed: true},
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error { return nil }

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	// Should complete normally without secrets
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// No secrets should be injected
	if len(capturedEnv) != 0 {
		t.Errorf("Expected no env vars, got %v", capturedEnv)
	}
}

func TestRunner_Run_CorruptedBlobLogsWarningAndContinues(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	t.Setenv("SPINNER_SECRET_PASSPHRASE", "test-passphrase")

	// Create a dummy file at the blob path so os.Stat succeeds
	corruptBlobPath := filepath.Join(tmpDir, "corrupt.enc")
	if err := os.WriteFile(corruptBlobPath, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}

	oldBlobPath := secretsBlobPath

	defer func() { secretsBlobPath = oldBlobPath }()

	secretsBlobPath = corruptBlobPath

	// Override decryptBlobFunc to simulate corruption
	oldDecryptBlob := decryptBlobFunc

	defer func() { decryptBlobFunc = oldDecryptBlob }()

	decryptBlobFunc = func(path, passphrase string) (map[string]string, error) {
		return nil, fmt.Errorf("decrypting secrets blob: cipher: message authentication failed")
	}

	// Override osUnsetenv to track it's NOT called on failure
	var unsetCalled bool

	oldOsUnsetenv := osUnsetenv

	defer func() { osUnsetenv = oldOsUnsetenv }()

	osUnsetenv = func(key string) error {
		unsetCalled = true
		return nil
	}

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 1,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{}
	runner := NewRunner(config, state, statePath)

	var capturedEnv []string

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		capturedEnv = env

		return &mockExecutor{
			result: &agent.Result{Completed: true},
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error { return nil }

	ctx := context.Background()
	exitCode := runner.Run(ctx)

	// Should complete normally despite corrupted blob
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// No secrets injected
	if len(capturedEnv) != 0 {
		t.Errorf("Expected no env vars, got %v", capturedEnv)
	}

	// Passphrase should NOT be unset (decryption failed, so we don't touch env)
	if unsetCalled {
		t.Error("Expected osUnsetenv NOT to be called on decryption failure")
	}
}

func TestRunner_Run_BlobNotDeletedAfterDecryption(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	testSecrets := map[string]string{"MY_SECRET": "value"}
	passphrase := "test-passphrase"
	blobPath := createTestBlob(t, testSecrets, passphrase)

	oldBlobPath := secretsBlobPath

	defer func() { secretsBlobPath = oldBlobPath }()

	secretsBlobPath = blobPath

	oldOsUnsetenv := osUnsetenv

	defer func() { osUnsetenv = oldOsUnsetenv }()

	osUnsetenv = func(key string) error { return nil }

	t.Setenv("SPINNER_SECRET_PASSPHRASE", passphrase)

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 1,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{}
	runner := NewRunner(config, state, statePath)

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		return &mockExecutor{
			result: &agent.Result{Completed: true},
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error { return nil }

	ctx := context.Background()
	runner.Run(ctx)

	// Blob file should still exist (not deleted — needed for inception)
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		t.Error("Expected blob file to still exist after decryption")
	}
}

func TestRunner_Run_SecretEnvSorted(t *testing.T) {
	// Verify that env vars are consistently ordered (map iteration is random)
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	testSecrets := map[string]string{
		"Z_SECRET": "z-val",
		"A_SECRET": "a-val",
		"M_SECRET": "m-val",
	}
	passphrase := "test-passphrase"
	blobPath := createTestBlob(t, testSecrets, passphrase)

	oldBlobPath := secretsBlobPath

	defer func() { secretsBlobPath = oldBlobPath }()

	secretsBlobPath = blobPath

	oldOsUnsetenv := osUnsetenv

	defer func() { osUnsetenv = oldOsUnsetenv }()

	osUnsetenv = func(key string) error { return nil }

	t.Setenv("SPINNER_SECRET_PASSPHRASE", passphrase)

	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 1,
		Branch:        "main",
		LogDir:        tmpDir,
	}

	state := &State{}
	runner := NewRunner(config, state, statePath)

	var capturedEnv []string

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string, additionalWriter io.Writer, env []string) agent.Executor {
		capturedEnv = env

		return &mockExecutor{
			result: &agent.Result{Completed: true},
		}
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error { return nil }

	ctx := context.Background()
	runner.Run(ctx)

	// Extract just the secret keys (not SPINNER_SECRET_PASSPHRASE or SPINNER_SECRET_STORE)
	var secretKeys []string

	for _, e := range capturedEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && !strings.HasPrefix(parts[0], "SPINNER_SECRET_") {
			secretKeys = append(secretKeys, parts[0])
		}
	}

	if !sort.StringsAreSorted(secretKeys) {
		t.Errorf("Expected secret env keys to be sorted, got %v", secretKeys)
	}
}

func TestWaitForRateLimit_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()

	waitForRateLimit(ctx)

	elapsed := time.Since(start)

	// Should exit quickly due to cancellation, not wait full time
	if elapsed > 1*time.Second {
		t.Errorf("waitForRateLimit took too long: %v", elapsed)
	}
}
