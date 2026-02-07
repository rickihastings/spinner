package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/compute/metadata"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
	"github.com/rickihastings/spinner/internal/logs"
)

const (
	// RateLimitWaitSeconds is how long to wait when rate limited (61 minutes)
	RateLimitWaitSeconds = 3660
)

// Function variables for testing
var (
	executorFactory = func(logPath string, additionalWriter io.Writer) agent.Executor {
		return claude.NewExecutor(&claude.ExecutorConfig{
			LogPath:          logPath,
			AdditionalWriter: additionalWriter,
		})
	}
	pushChangesFunc    = PushChanges
	gcsSinkFactory     = defaultGCSSinkFactory
	gcsObjectWriterNew = logs.NewGCSObjectWriter
	isRunningOnGCE     = metadata.OnGCE
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

	// Create GCS log sink if running inside a GCP VM with a log bucket configured.
	// Use io.Writer type to avoid Go's interface nil pitfall: a nil *GCSSink
	// stored in an io.Writer interface would not compare equal to nil.
	var additionalWriter io.Writer

	sink, cleanup := gcsSinkFactory(ctx)
	if sink != nil {
		additionalWriter = sink

		defer cleanup()
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
		executor := executorFactory(logPath, additionalWriter)

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

// defaultGCSSinkFactory creates a GCS sink when running on a GCE VM with
// SPINNER_LOG_BUCKET set. The metadata server is queried first to confirm
// this is actually a GCP environment, preventing accidental activation when
// the env var leaks into a non-GCP context (e.g. Docker).
// Returns (nil, nil) if not on GCE, the env var is unset, or init fails.
func defaultGCSSinkFactory(ctx context.Context) (*logs.GCSSink, func()) {
	if !isRunningOnGCE() {
		return nil, nil
	}

	bucket := os.Getenv("SPINNER_LOG_BUCKET")
	if bucket == "" {
		return nil, nil
	}

	instanceName := os.Getenv("SPINNER_INSTANCE_NAME")
	if instanceName == "" {
		fmt.Fprintf(os.Stderr, "Warning: SPINNER_LOG_BUCKET set but SPINNER_INSTANCE_NAME empty, skipping GCS log sink\n")
		return nil, nil
	}

	objectWriter, err := gcsObjectWriterNew(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create GCS client for log streaming: %v\n", err)
		return nil, nil
	}

	object := instanceName + "/logs/raw.log"
	sink := logs.NewGCSSink(ctx, objectWriter, bucket, object)

	fmt.Printf("GCS log streaming enabled: gs://%s/%s\n", bucket, object)

	cleanup := func() {
		_ = sink.Close()
		_ = objectWriter.Close()
	}

	return sink, cleanup
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
