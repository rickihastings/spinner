package exec

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
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

	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
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

	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
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

	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
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

	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
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

	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
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

	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
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

	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
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
