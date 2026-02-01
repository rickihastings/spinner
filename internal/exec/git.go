package exec

import (
	"context"
	"fmt"
	"os/exec"
)

// PushChanges attempts to push changes to the remote repository.
// It first tries `git push`, and if that fails (e.g., no upstream branch),
// it tries `git push -u origin <branch>` to set the tracking branch.
// Returns error only for logging purposes - push failures are non-blocking.
func PushChanges(ctx context.Context, branch string) error {
	// Try simple push first
	cmd := exec.CommandContext(ctx, "git", "push")
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If simple push failed, try setting upstream
	cmd = exec.CommandContext(ctx, "git", "push", "-u", "origin", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push to remote: %w", err)
	}

	return nil
}
