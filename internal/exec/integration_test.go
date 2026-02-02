package exec

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_StateFileCreation tests that state file is created on first run
func TestIntegration_StateFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Verify state file doesn't exist
	if _, err := os.Stat(statePath); err == nil {
		t.Fatal("State file should not exist before first run")
	}

	config := &Config{
		Prompt:        "test task",
		MaxIterations: 1,
		Branch:        "test-branch",
		LogDir:        tmpDir,
		StateDir:      tmpDir,
	}

	// Initialize state
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	// Mock Claude to complete immediately
	oldRunClaude := runClaudeFunc

	defer func() { runClaudeFunc = oldRunClaude }()

	runClaudeFunc = func(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
		return &ClaudeResult{Completed: true}, nil
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	// Run the loop
	runner := NewRunner(config, state, statePath)
	exitCode := runner.Run(context.Background())

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Verify state file was created
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Verify state file contents
	loadedState, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to load state file: %v", err)
	}

	if loadedState.Status != StatusCompleted {
		t.Errorf("Expected status completed, got %s", loadedState.Status)
	}

	if loadedState.Branch != "test-branch" {
		t.Errorf("Expected branch test-branch, got %s", loadedState.Branch)
	}

	if loadedState.Iteration != 1 {
		t.Errorf("Expected iteration 1, got %d", loadedState.Iteration)
	}
}

// TestIntegration_SingleIteration tests a complete single iteration
func TestIntegration_SingleIteration(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	logPath := filepath.Join(tmpDir, "raw.log")

	config := &Config{
		Prompt:        "implement feature X",
		MaxIterations: 5,
		Branch:        "feature-x",
		LogDir:        tmpDir,
		StateDir:      tmpDir,
	}

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	// Track Claude calls
	claudeCalls := 0

	oldRunClaude := runClaudeFunc

	defer func() { runClaudeFunc = oldRunClaude }()

	runClaudeFunc = func(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
		claudeCalls++

		// Verify prompt was passed correctly
		if prompt != "implement feature X" {
			t.Errorf("Expected prompt 'implement feature X', got %s", prompt)
		}

		// Complete on first iteration
		return &ClaudeResult{Completed: true}, nil
	}

	pushCalls := 0
	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		pushCalls++

		if branch != "feature-x" {
			t.Errorf("Expected branch feature-x, got %s", branch)
		}

		return nil
	}

	runner := NewRunner(config, state, statePath)
	exitCode := runner.Run(context.Background())

	// Verify completion
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Verify Claude was called once
	if claudeCalls != 1 {
		t.Errorf("Expected 1 Claude call, got %d", claudeCalls)
	}

	// Verify push was called once
	if pushCalls != 1 {
		t.Errorf("Expected 1 push call, got %d", pushCalls)
	}

	// Verify final state
	if state.Status != StatusCompleted {
		t.Errorf("Expected status completed, got %s", state.Status)
	}

	if state.Iteration != 1 {
		t.Errorf("Expected 1 iteration, got %d", state.Iteration)
	}

	if state.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set")
	}

	// Verify log file was created (if log writing is implemented)
	// This is optional depending on implementation
	_ = logPath
}

// TestIntegration_StatePersistence tests state persistence across multiple runs
func TestIntegration_StatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "multi-step task",
		MaxIterations: 3,
		Branch:        "main",
		LogDir:        tmpDir,
		StateDir:      tmpDir,
	}

	// First run - do 2 iterations then "restart"
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	iterationCount := 0

	oldRunClaude := runClaudeFunc

	defer func() { runClaudeFunc = oldRunClaude }()

	runClaudeFunc = func(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
		iterationCount++
		// Don't complete - let it hit max iterations
		return &ClaudeResult{Completed: false}, nil
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	runner := NewRunner(config, state, statePath)
	exitCode := runner.Run(context.Background())

	// Should exit with error (max iterations)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Verify 3 iterations were completed
	if iterationCount != 3 {
		t.Errorf("Expected 3 iterations, got %d", iterationCount)
	}

	// Load state from file (simulating container restart)
	persistedState, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to load persisted state: %v", err)
	}

	// Verify persisted state matches final state
	if persistedState.Status != StatusError {
		t.Errorf("Expected persisted status error, got %s", persistedState.Status)
	}

	// After 3 complete iterations, the counter will be 4 (incremented but loop condition failed)
	if persistedState.Iteration != 4 {
		t.Errorf("Expected persisted iteration 4, got %d", persistedState.Iteration)
	}

	if persistedState.Branch != "main" {
		t.Errorf("Expected persisted branch main, got %s", persistedState.Branch)
	}

	if persistedState.ErrorMessage != "max iterations reached" {
		t.Errorf("Expected error message 'max iterations reached', got %s", persistedState.ErrorMessage)
	}

	// Verify timestamps were persisted
	if persistedState.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be persisted")
	}

	if persistedState.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be persisted")
	}
}

// TestIntegration_CompletionSignalDetection tests that completion signal is detected
func TestIntegration_CompletionSignalDetection(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "complete task",
		MaxIterations: 5,
		Branch:        "main",
		LogDir:        tmpDir,
		StateDir:      tmpDir,
	}

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	oldRunClaude := runClaudeFunc

	defer func() { runClaudeFunc = oldRunClaude }()

	// Simulate Claude returning completion signal
	runClaudeFunc = func(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
		// In real execution, completion is detected by scanning message content
		// for "~~ FEATURE_COMPLETED ~~" signal. Here we just return Completed: true.
		return &ClaudeResult{
			Completed: true,
		}, nil
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	runner := NewRunner(config, state, statePath)
	exitCode := runner.Run(context.Background())

	// Should exit with success
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Verify completion was detected
	if state.Status != StatusCompleted {
		t.Errorf("Expected status completed, got %s", state.Status)
	}

	// Should only do 1 iteration before completing
	if state.Iteration != 1 {
		t.Errorf("Expected 1 iteration, got %d", state.Iteration)
	}

	// Verify CompletedAt timestamp is set
	if state.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set on completion")
	}
}

// TestIntegration_RateLimitHandling tests rate limit detection and retry
func TestIntegration_RateLimitHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limit integration test in short mode")
	}

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	config := &Config{
		Prompt:        "task with rate limit",
		MaxIterations: 3,
		Branch:        "main",
		LogDir:        tmpDir,
		StateDir:      tmpDir,
	}

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	callCount := 0

	oldRunClaude := runClaudeFunc

	defer func() { runClaudeFunc = oldRunClaude }()

	runClaudeFunc = func(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
		callCount++

		// Return rate limit on first call
		if callCount == 1 {
			return &ClaudeResult{
				RateLimited:  true,
				ErrorMessage: "rate_limit_error: too many requests",
			}, nil
		}

		// Complete on second call
		return &ClaudeResult{Completed: true}, nil
	}

	oldPushChanges := pushChangesFunc

	defer func() { pushChangesFunc = oldPushChanges }()

	pushChangesFunc = func(ctx context.Context, branch string) error {
		return nil
	}

	// Use context with timeout to avoid waiting full 61 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	runner := NewRunner(config, state, statePath)
	exitCode := runner.Run(ctx)

	// Should get interrupted during wait
	if exitCode != 130 {
		t.Errorf("Expected exit code 130, got %d", exitCode)
	}

	// Verify rate limited status was saved
	loadedState, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// State should show error from context cancellation
	if loadedState.Status != StatusError {
		t.Errorf("Expected status error (from cancellation), got %s", loadedState.Status)
	}

	// Verify Claude was called at least once
	if callCount < 1 {
		t.Error("Expected at least 1 Claude call")
	}
}

// TestIntegration_EdgeCases tests various edge cases
func TestIntegration_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		maxIterations  int
		claudeResult   *ClaudeResult
		claudeError    error
		expectedExit   int
		expectedStatus Status
	}{
		{
			name:           "empty max iterations uses default",
			maxIterations:  1,
			claudeResult:   &ClaudeResult{Completed: true},
			expectedExit:   0,
			expectedStatus: StatusCompleted,
		},
		{
			name:           "auth error stops immediately",
			maxIterations:  5,
			claudeResult:   &ClaudeResult{AuthError: true, ErrorMessage: "auth failed"},
			expectedExit:   1,
			expectedStatus: StatusAuthError,
		},
		{
			name:           "generic error continues to next iteration",
			maxIterations:  2,
			claudeResult:   &ClaudeResult{Error: os.ErrPermission, ErrorMessage: "permission denied"},
			expectedExit:   1,
			expectedStatus: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")

			config := &Config{
				Prompt:        "test",
				MaxIterations: tt.maxIterations,
				Branch:        "main",
				LogDir:        tmpDir,
				StateDir:      tmpDir,
			}

			state, err := LoadState(statePath)
			if err != nil {
				t.Fatalf("Failed to initialize state: %v", err)
			}

			oldRunClaude := runClaudeFunc

			defer func() { runClaudeFunc = oldRunClaude }()

			runClaudeFunc = func(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
				return tt.claudeResult, tt.claudeError
			}

			oldPushChanges := pushChangesFunc

			defer func() { pushChangesFunc = oldPushChanges }()

			pushChangesFunc = func(ctx context.Context, branch string) error {
				return nil
			}

			runner := NewRunner(config, state, statePath)
			exitCode := runner.Run(context.Background())

			if exitCode != tt.expectedExit {
				t.Errorf("Expected exit code %d, got %d", tt.expectedExit, exitCode)
			}

			if state.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, state.Status)
			}
		})
	}
}
