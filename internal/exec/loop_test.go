package exec

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
)

// mockExecutor is a test helper that implements agent.Executor interface
type mockExecutor struct {
	result *agent.Result
	err    error
}

func (m *mockExecutor) Execute(ctx context.Context, prompt string) (<-chan agent.Event, error) {
	// Not used in loop tests
	return nil, nil
}

func (m *mockExecutor) ExecuteAndCollect(ctx context.Context, prompt string) (*agent.Result, error) {
	return m.result, m.err
}

func TestNewRunner(t *testing.T) {
	config := &Config{
		Prompt:        "test prompt",
		MaxIterations: 5,
		Branch:        "main",
	}

	state := &State{
		Status:    StatusRunning,
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
		Status:    StatusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor factory to return no completion
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string) agent.Executor {
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
	if state.Status != StatusError {
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

	if loadedState.Status != StatusError {
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
		Status:    StatusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor factory to return completion on first iteration
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string) agent.Executor {
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

	if state.Status != StatusCompleted {
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
		Status:    StatusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string) agent.Executor {
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

	if state.Status != StatusAuthError {
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
		Status:    StatusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	// Mock executor factory to return rate limit on first call, then completion
	callCount := 0
	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string) agent.Executor {
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
		Status:    StatusRunning,
		Iteration: 0,
	}

	runner := NewRunner(config, state, statePath)

	oldExecutorFactory := executorFactory

	defer func() { executorFactory = oldExecutorFactory }()

	executorFactory = func(logPath string) agent.Executor {
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

	if state.Status != StatusError {
		t.Errorf("Expected status error, got %s", state.Status)
	}

	if state.ErrorMessage != "interrupted by user" {
		t.Errorf("Expected error message 'interrupted by user', got %s", state.ErrorMessage)
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
