package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
)

const (
	// RateLimitWaitSeconds is how long to wait when rate limited (61 minutes)
	RateLimitWaitSeconds = 3660

	// maxConsecutiveErrors is the number of consecutive errors before the loop
	// stops. This prevents burning through all iterations when Claude CLI is
	// broken (e.g., not installed, crashing immediately).
	maxConsecutiveErrors = 3
)

// StateSyncFunc is called after each local state save to sync state to a
// remote store (e.g., GCS). It receives the path to the local state file.
// Errors are logged as warnings but do not fail the loop.
type StateSyncFunc func(statePath string)

// LogSinkFactory creates an optional io.Writer for streaming logs to a remote
// store. Returns (nil, nil) if not applicable. The cleanup function should be
// called when the writer is no longer needed.
type LogSinkFactory func(ctx context.Context) (io.Writer, func())

// StateSyncFactory creates an optional function for syncing state to a remote
// store. Returns nil if not applicable.
type StateSyncFactory func(ctx context.Context) StateSyncFunc

// Function variables for testing
var (
	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
		return claude.NewExecutor(&claude.ExecutorConfig{
			LogPath:          logPath,
			AdditionalWriter: additionalWriter,
		})
	}
	pushChangesFunc = pushChanges
)

// Runner executes the main iteration loop.
type Runner struct {
	config           *Config
	state            *State
	statePath        string
	stateSync        StateSyncFunc
	logSinkFactory   LogSinkFactory
	stateSyncFactory StateSyncFactory
}

// RunnerOption configures optional Runner behavior.
type RunnerOption func(*Runner)

// WithLogSinkFactory sets a factory for creating a remote log sink.
func WithLogSinkFactory(f LogSinkFactory) RunnerOption {
	return func(r *Runner) {
		r.logSinkFactory = f
	}
}

// WithStateSyncFactory sets a factory for creating a remote state sync function.
func WithStateSyncFactory(f StateSyncFactory) RunnerOption {
	return func(r *Runner) {
		r.stateSyncFactory = f
	}
}

// NewRunner creates a new Runner with the given config and state.
func NewRunner(config *Config, state *State, statePath string, opts ...RunnerOption) *Runner {
	r := &Runner{
		config:    config,
		state:     state,
		statePath: statePath,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// saveState saves state locally and, if configured, syncs to a remote store.
// It collects system metrics (CPU/memory from /proc) before saving.
func (r *Runner) saveState() error {
	r.state.CPUPercent, r.state.MemoryPercent = collectSystemMetrics()

	err := saveState(r.statePath, r.state)
	if err == nil && r.stateSync != nil {
		r.stateSync(r.statePath)
	}

	return err
}

// Run executes the main iteration loop.
// Returns exit code: 0 for completion, 1 for auth error, 1 for max iterations reached.
func (r *Runner) Run(ctx context.Context) int {
	fmt.Printf("Starting Ralph loop with prompt: %s\n", r.config.Prompt)
	fmt.Printf("Max iterations: %d\n", r.config.MaxIterations)

	// Set initial state
	r.state.Branch = r.config.Branch

	// Set up remote state sync if a factory is configured.
	if r.stateSyncFactory != nil {
		r.stateSync = r.stateSyncFactory(ctx)
	}

	r.state.Status = statusRunning
	if err := r.saveState(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save initial state: %v\n", err)
	}

	// Create remote log sink if a factory is configured.
	// Use io.Writer type to avoid Go's interface nil pitfall: a nil concrete
	// type stored in an io.Writer interface would not compare equal to nil.
	var additionalWriter io.Writer

	if r.logSinkFactory != nil {
		sink, cleanup := r.logSinkFactory(ctx)
		if sink != nil {
			additionalWriter = sink

			defer cleanup()
		}
	}

	// Truncate the log file once at session start so previous runs don't
	// accumulate. Individual iterations append to this file.
	if r.config.LogDir != "" {
		logPath := filepath.Join(r.config.LogDir, "raw.log")
		if err := os.Truncate(logPath, 0); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: failed to truncate log file: %v\n", err)
		}
	}

	consecutiveErrors := 0

	for r.state.Iteration = 1; r.state.Iteration <= r.config.MaxIterations; r.state.Iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			fmt.Println("\n⚠️  Loop interrupted by user (Ctrl+C)")

			r.state.Status = statusError
			r.state.ErrorMessage = "interrupted by user"
			_ = r.saveState()

			return 130
		default:
		}

		fmt.Printf("\n🔁 Iteration %d/%d\n", r.state.Iteration, r.config.MaxIterations)

		// Save state before iteration
		if err := r.saveState(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
		}

		// Build log file path
		logPath := ""
		if r.config.LogDir != "" {
			logPath = filepath.Join(r.config.LogDir, "raw.log")
		}

		// Run Claude
		executor := executorFactory(logPath, additionalWriter)

		result, err := executor.ExecuteAndCollect(ctx, r.config.Prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running Claude: %v\n", err)

			r.state.Status = statusError
			r.state.ErrorMessage = err.Error()
			_ = r.saveState()

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

			r.state.Status = statusAuthError
			r.state.ErrorMessage = result.ErrorMessage
			_ = r.saveState()

			return 1
		}

		// Check for feature completion
		if result.Completed {
			fmt.Println("\n\n🎯 ALL TASKS COMPLETE.")

			r.state.Status = statusCompleted
			r.state.CompletedAt = time.Now()
			_ = r.saveState()

			return 0
		}

		// Check for rate limiting
		if result.RateLimited {
			r.state.Status = statusRateLimited
			r.state.ErrorMessage = result.ErrorMessage
			_ = r.saveState()

			waitForRateLimit(ctx)

			// Reset status and don't increment iteration (redo this iteration)
			r.state.Status = statusRunning
			r.state.ErrorMessage = ""
			r.state.Iteration--

			continue
		}

		// Check for other errors
		if result.Error != nil {
			consecutiveErrors++

			fmt.Fprintf(os.Stderr, "\n⚠️  Error during iteration: %v\n", result.Error)

			if consecutiveErrors >= maxConsecutiveErrors {
				fmt.Fprintf(os.Stderr, "\n❌ %d consecutive errors detected, stopping loop\n", maxConsecutiveErrors)

				r.state.Status = statusError
				r.state.ErrorMessage = fmt.Sprintf("stopped after %d consecutive errors: %s", maxConsecutiveErrors, result.ErrorMessage)
				_ = r.saveState()

				return 1
			}

			r.state.Status = statusError
			r.state.ErrorMessage = result.ErrorMessage
			_ = r.saveState()
			// Continue to next iteration instead of exiting
		} else {
			consecutiveErrors = 0
		}

		fmt.Println("\n✓ Iteration complete.")
	}

	// Max iterations reached
	fmt.Printf("\n⚠️  Max iterations (%d) reached\n", r.config.MaxIterations)
	r.state.Status = statusError
	r.state.ErrorMessage = "max iterations reached"
	_ = r.saveState()

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
