package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
)

// destroyCmd is the production destroy command using the default provider factory.
var destroyCmd = NewDestroyCommand(defaultFactory)

func init() {
	rootCmd.AddCommand(destroyCmd)
}

// NewDestroyCommand creates a new destroy command with the given Factory.
// This constructor enables dependency injection for testing.
func NewDestroyCommand(f *provider.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy <instance-name>...",
		Short: "Destroy one or more instances and clean up their state",
		Long: `Destroy one or more instances and clean up their state

USAGE:
  spinner destroy <instance-name>... [--backend docker|gcp] [options]

DESCRIPTION:
  The destroy command forcefully removes instances regardless of their current
  state (running, stopped, or exited). It also removes the host-side state
  directory (~/.spinner/<instance-name>/) including logs and state files.

  Multiple instance names can be provided. The command will process all
  instances and report per-instance success or failure.

EXAMPLES:
  # Destroy a single Docker container
  spinner destroy my-container

  # Destroy multiple instances
  spinner destroy instance1 instance2 instance3

  # Destroy a GCP VM instance
  spinner destroy my-instance --backend gcp --project my-proj --zone us-central1-a --state-bucket my-bucket`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backend, err := resolveAndValidateBackend(cmd)
			if err != nil {
				return err
			}

			p, err := f.Create(backend)
			if err != nil {
				return err
			}

			ctx := context.Background()
			var failures []string

			for _, instanceName := range args {
				// Check if instance exists
				status, err := p.Status(ctx, instanceName)
				if err != nil || status == provider.InstanceStatusNone {
					fmt.Fprintf(cmd.ErrOrStderr(), "✗ Instance '%s' not found\n", instanceName)
					failures = append(failures, instanceName)
					continue
				}

				// Remove the instance
				if err := p.Remove(ctx, instanceName); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "✗ Failed to destroy instance '%s': %s\n", instanceName, err.Error())
					failures = append(failures, instanceName)
					continue
				}

				// Remove state directory
				homeDir, err := os.UserHomeDir()
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Warning: Failed to get home directory for '%s': %s\n", instanceName, err.Error())
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Instance '%s' destroyed (state directory not cleaned)\n", instanceName)
					continue
				}

				stateDir := filepath.Join(homeDir, ".spinner", instanceName)
				if err := os.RemoveAll(stateDir); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Warning: Failed to remove state directory for '%s': %s\n", instanceName, err.Error())
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Instance '%s' destroyed (state directory not cleaned)\n", instanceName)
					continue
				}

				fmt.Fprintf(cmd.OutOrStdout(), "✓ Instance '%s' destroyed\n", instanceName)
			}

			// Return aggregate error if any instance failed
			if len(failures) > 0 {
				return fmt.Errorf("failed to destroy %d of %d instance(s)", len(failures), len(args))
			}

			return nil
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
