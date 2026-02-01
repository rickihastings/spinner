package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rickihastings/spinner/internal/exec"
	"github.com/spf13/cobra"
)

// NewExecCommand creates a new exec command.
// This command runs inside Docker containers and executes the iteration loop.
func NewExecCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute the autonomous iteration loop (runs inside containers)",
		Long: `Execute the autonomous iteration loop inside a Docker container.

This command is designed to run inside Docker containers created by 'spinner spin'.
It reads configuration from environment variables and manages an iteration loop
that interacts with Claude CLI to complete tasks.

ENVIRONMENT VARIABLES:
  PROMPT             Task prompt for the iteration loop (required)
  MAX_ITERATIONS     Maximum number of iterations (required)
  BRANCH             Git branch name (optional)
  LOG_DIR            Directory for log files (optional)
  CONTAINER_NAME     Container name for state file location (optional)

STATE MANAGEMENT:
  State is persisted to ~/.spinner/{CONTAINER_NAME}/state.json
  This allows iteration progress to survive container restarts.

EXAMPLES:
  # Typically called automatically by container startup script
  spinner exec

  # Can be called manually with environment variables
  PROMPT="Fix bug" MAX_ITERATIONS=10 spinner exec`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration from environment variables
			config, err := exec.LoadConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
				return err
			}

			// Determine state file path
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
				return err
			}

			// Use container name if available, otherwise use "default"
			containerDir := "default"
			if config.ContainerName != "" {
				containerDir = config.ContainerName
			}

			statePath := filepath.Join(home, ".spinner", containerDir, "state.json")

			// Load or initialize state
			state, err := exec.LoadState(statePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
				return err
			}

			// Create context with signal handling for Ctrl+C
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Set up signal handling
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				fmt.Println("\nReceived interrupt signal...")
				cancel()
			}()

			// Create runner and execute loop
			runner := exec.NewRunner(config, state, statePath)
			exitCode := runner.Run(ctx)

			// Exit with the appropriate code
			os.Exit(exitCode)

			return nil
		},
	}

	return cmd
}
