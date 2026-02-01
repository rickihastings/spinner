package exec

import (
	"context"
	"testing"
)

func TestPushChanges_ValidBranchName(t *testing.T) {
	// This test validates that PushChanges doesn't panic with valid inputs
	// Actual git operations are tested in integration tests
	ctx := context.Background()
	branch := "main"

	// The function should not panic even if git push fails
	// (it returns an error but doesn't panic)
	_ = PushChanges(ctx, branch)
}

func TestPushChanges_EmptyBranch(t *testing.T) {
	// Test with empty branch name
	ctx := context.Background()
	branch := ""

	// Should not panic even with empty branch
	_ = PushChanges(ctx, branch)
}

func TestPushChanges_SpecialCharacters(t *testing.T) {
	// Test with branch name containing special characters
	ctx := context.Background()
	branch := "feature/add-exec-command"

	// Should handle branch names with slashes
	_ = PushChanges(ctx, branch)
}

func TestPushChanges_ContextCancellation(t *testing.T) {
	// Test that cancelled context is handled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	branch := "main"
	err := PushChanges(ctx, branch)

	// Should return error when context is cancelled
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
	}
}
