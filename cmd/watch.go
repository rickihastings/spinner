package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rickihastings/spinner/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

			return performWatch(context.Background(), p, args[0])
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
func gatherWatchContext(ctx context.Context, p provider.Provider, instanceName string) tui.WatchContext {
	wctx := tui.WatchContext{}

	// Get instance metadata from provider
	if metadata, err := p.GetInstanceMetadata(ctx, instanceName); err == nil && metadata != nil {
		wctx.Environment = metadata.Backend
		wctx.ContainerID = metadata.InstanceID
		wctx.ImageID = metadata.ImageID
		wctx.Agent = metadata.Agent
		wctx.MaxIterations = metadata.MaxIterations
		wctx.Branch = metadata.Branch
	}

	return wctx
}

// performWatch executes the watch workflow for an instance.
// Used by both the standalone watch command and spin --watch.
func performWatch(ctx context.Context, p provider.Provider, containerName string) error {
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
	uiContext := gatherWatchContext(ctx, p, containerName)
	uiContext.HeaderVisible = viper.GetBool("watch-header")

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
