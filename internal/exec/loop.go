package exec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
)

const (
	// RateLimitWaitSeconds is how long to wait when rate limited (61 minutes)
	RateLimitWaitSeconds = 3660
)

// Function variables for testing
var (
	executorFactory = func(logPath string) agent.Executor {
		return claude.NewExecutor(&claude.ExecutorConfig{
			LogPath: logPath,
		})
	}
	pushChangesFunc = PushChanges
)

// Runner executes the main iteration loop.
type Runner struct {
	config    *Config
	state     *State
	statePath string
}

// NewRunner creates a new Runner with the given config and state.
func NewRunner(config *Config, state *State, statePath string) *Runner {
	return &Runner{
		config:    config,
		state:     state,
		statePath: statePath,
	}
}

// Run executes the main iteration loop.
// Returns exit code: 0 for completion, 1 for auth error, 1 for max iterations reached.
func (r *Runner) Run(ctx context.Context) int {
	fmt.Printf("Starting Ralph loop with prompt: %s\n", r.config.Prompt)
	fmt.Printf("Max iterations: %d\n", r.config.MaxIterations)

	// Set initial state
	r.state.Branch = r.config.Branch

	r.state.Status = StatusRunning
	if err := SaveState(r.statePath, r.state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save initial state: %v\n", err)
	}

	for r.state.Iteration = 1; r.state.Iteration <= r.config.MaxIterations; r.state.Iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			fmt.Println("\n⚠️  Loop interrupted by user (Ctrl+C)")

			r.state.Status = StatusError
			r.state.ErrorMessage = "interrupted by user"
			_ = SaveState(r.statePath, r.state)

			return 130
		default:
		}

		fmt.Printf("\n🔁 Iteration %d/%d\n", r.state.Iteration, r.config.MaxIterations)

		// Save state before iteration
		if err := SaveState(r.statePath, r.state); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
		}

		// Build log file path
		logPath := ""
		if r.config.LogDir != "" {
			logPath = filepath.Join(r.config.LogDir, "raw.log")
		}

		// Run Claude
		executor := executorFactory(logPath)

		result, err := executor.ExecuteAndCollect(ctx, r.config.Prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running Claude: %v\n", err)

			r.state.Status = StatusError
			r.state.ErrorMessage = err.Error()
			_ = SaveState(r.statePath, r.state)

			return 1
		}

		// Push changes to remote
		if r.config.Branch != "" {
			fmt.Println("Pushing changes to remote...")

			if err := pushChangesFunc(ctx, r.config.Branch); err != nil {
				fmt.Printf("⚠️  Push failed: %v\n", err)
				fmt.Println("⚠️  Continuing anyway...")
			} else {
				fmt.Println("✓ Push successful")
			}
		}

		// Check for authentication error first
		if result.AuthError {
			fmt.Println("\n\n❌ Authentication error detected - please run /login")

			r.state.Status = StatusAuthError
			r.state.ErrorMessage = result.ErrorMessage
			_ = SaveState(r.statePath, r.state)

			return 1
		}

		// Check for feature completion
		if result.Completed {
			fmt.Println("\n\n🎯 ALL TASKS COMPLETE.")

			r.state.Status = StatusCompleted
			r.state.CompletedAt = time.Now()
			_ = SaveState(r.statePath, r.state)

			return 0
		}

		// Check for rate limiting
		if result.RateLimited {
			r.state.Status = StatusRateLimited
			r.state.ErrorMessage = result.ErrorMessage
			_ = SaveState(r.statePath, r.state)

			waitForRateLimit(ctx)

			// Reset status and don't increment iteration (redo this iteration)
			r.state.Status = StatusRunning
			r.state.ErrorMessage = ""
			r.state.Iteration--

			continue
		}

		// Check for other errors
		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "\n⚠️  Error during iteration: %v\n", result.Error)

			r.state.Status = StatusError
			r.state.ErrorMessage = result.ErrorMessage
			_ = SaveState(r.statePath, r.state)
			// Continue to next iteration instead of exiting
		}

		fmt.Println("\n✓ Iteration complete.")
	}

	// Max iterations reached
	fmt.Printf("\n⚠️  Max iterations (%d) reached\n", r.config.MaxIterations)
	r.state.Status = StatusError
	r.state.ErrorMessage = "max iterations reached"
	_ = SaveState(r.statePath, r.state)

	return 1
}

// waitForRateLimit waits for the rate limit period with a countdown.
func waitForRateLimit(ctx context.Context) {
	fmt.Println("\n⚠️  Rate limit detected - waiting 61 minutes...")

	remaining := RateLimitWaitSeconds

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for remaining > 0 {
		select {
		case <-ctx.Done():
			fmt.Println("\n⚠️  Wait interrupted by user")
			return
		case <-ticker.C:
			remaining--
			// Print countdown every 60 seconds
			if remaining%60 == 0 {
				fmt.Printf("⏳ %d minutes remaining...\n", remaining/60)
			}
		}
	}

	fmt.Println("\n✅ Wait complete - resuming...")
}
