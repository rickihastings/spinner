package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rickihastings/spinner/internal/tui"
	"github.com/spf13/cobra"
)

// watchCmd is the production watch command using the default provider factory.
var watchCmd = NewWatchCommand(defaultFactory)

func init() {
	rootCmd.AddCommand(watchCmd)
}

// NewWatchCommand creates a new watch command with the given Factory.
// This constructor enables dependency injection for testing.
func NewWatchCommand(f *provider.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <instance-name>",
		Short: "Monitor instance logs and metrics in real-time",
		Long: `Monitor instance logs and metrics in real-time using a terminal UI

USAGE:
  spinner watch <instance-name> [--backend docker|gcp] [options]

DESCRIPTION:
  The watch command provides a real-time terminal UI for monitoring a running
  instance. It displays:

  - Instance status (running/stopped/exited)
  - CPU and memory usage metrics
  - Streaming logs with structured formatting

KEYBOARD SHORTCUTS:
  q         - Quit watch mode
  Ctrl+C    - Quit watch mode

EXAMPLES:
  # Watch a Docker container
  spinner watch my-container

  # Watch a GCP VM instance
  spinner watch my-instance --backend gcp --project my-proj --zone us-central1-a --state-bucket my-bucket`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backend, err := resolveAndValidateBackend(cmd)
			if err != nil {
				return err
			}

			p, err := f.Create(backend)
			if err != nil {
				return err
			}

			return PerformWatch(context.Background(), p, args[0])
		},
	}

	// General flags
	cmd.Flags().String(flagBackend, "", "Backend provider: docker, gcp (default: docker)")

	// GCP backend flags
	cmd.Flags().String(flagProject, "", "GCP project ID (GCP backend)")
	cmd.Flags().String(flagZone, "", "GCP zone (GCP backend)")
	cmd.Flags().String(flagStateBucket, "", "GCS bucket for state persistence (GCP backend)")

	return cmd
}

// gatherWatchContext collects execution context for the watch UI
func gatherWatchContext(_ context.Context, containerName string) tui.WatchContext {
	wctx := tui.WatchContext{
		Environment: "docker",
	}

	// Get git branch
	if branch := getGitBranch(); branch != "" {
		wctx.Branch = branch
	}

	// Get container ID
	if containerID := getContainerID(containerName); containerID != "" {
		wctx.ContainerID = containerID
	}

	// Get image ID
	if imageID := getImageID(containerName); imageID != "" {
		wctx.ImageID = imageID
	}

	// Get agent model from environment variable (set by container)
	if model := os.Getenv("ANTHROPIC_MODEL"); model != "" {
		wctx.Agent = model
	}

	// Get max iterations from environment variable (set by container)
	if maxIter := os.Getenv("MAX_ITERATIONS"); maxIter != "" {
		if val, err := strconv.Atoi(maxIter); err == nil {
			wctx.MaxIterations = val
		}
	}

	return wctx
}

// getGitBranch gets the current git branch name
func getGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getContainerID gets the container ID for a given container name
func getContainerID(containerName string) string {
	cmd := exec.Command("docker", "inspect", "--format={{.Id}}", containerName)

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getImageID gets the image ID for a given container name
func getImageID(containerName string) string {
	cmd := exec.Command("docker", "inspect", "--format={{.Image}}", containerName)

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// PerformWatch executes the watch workflow for an instance.
// This is exported so it can be used by both the standalone watch command
// and the spin command with --watch flag.
func PerformWatch(ctx context.Context, p provider.Provider, containerName string) error {
	// Check if instance exists using provider abstraction
	status, err := p.Status(ctx, containerName)
	if err != nil || status == provider.InstanceStatusNone {
		fmt.Fprintf(os.Stderr, "✗ Error: Instance '%s' not found\n", containerName)
		fmt.Fprintf(os.Stderr, "Tip: Check that the instance exists and the correct backend is selected\n")

		return fmt.Errorf("instance not found: %s", containerName)
	}

	// Create parser and formatter - the only place in cmd that imports claude
	parser := claude.NewParser()
	formatter := claude.NewFormatter()

	// Gather context information for the watch UI
	uiContext := gatherWatchContext(ctx, containerName)

	// Create channels for logs and metrics
	logCh := make(chan agent.Event, 100)
	metricsCh := make(chan provider.ContainerMetrics, 10)

	// Create TUI with the formatter and context
	ui := tui.NewWatchUI(containerName, formatter, uiContext)

	// Create context for coordinated shutdown
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create raw log channel for provider
	rawLogCh := make(chan string, 100)

	// Start log watcher in goroutine using provider abstraction
	go func() {
		defer close(rawLogCh)

		if err := p.WatchLogs(watchCtx, containerName, 100, rawLogCh); err != nil {
			// Only log error if not cancelled
			if watchCtx.Err() == nil {
				fmt.Fprintf(os.Stderr, "⚠ Log watcher error: %s\n", err.Error())
			}
		}
	}()

	// Parse raw log lines into Events
	go func() {
		defer close(logCh)

		for rawLine := range rawLogCh {
			if event := parser.ParseLine(rawLine); event != nil {
				select {
				case logCh <- *event:
				case <-watchCtx.Done():
					return
				}
			}
		}
	}()

	// Start metrics streaming in goroutine using provider abstraction
	go func() {
		defer close(metricsCh)

		if err := p.WatchMetrics(watchCtx, containerName, metricsCh); err != nil {
			// Only log error if not cancelled
			if watchCtx.Err() == nil {
				fmt.Fprintf(os.Stderr, "⚠ Metrics provider error: %s\n", err.Error())
			}
		}
	}()

	// Run TUI (blocks until quit)
	fmt.Printf("Starting watch mode for instance: %s\n", containerName)
	fmt.Printf("Press 'q' or Ctrl+C to quit\n\n")

	if err := ui.Run(logCh, metricsCh); err != nil {
		cancel()
		fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())

		return err
	}

	// Cancel context to stop goroutines
	cancel()

	return nil
}
