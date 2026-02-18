package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// setupCmd is the production setup command using the default provider factory.
var setupCmd = NewSetupCommand(defaultFactory)

func init() {
	rootCmd.AddCommand(setupCmd)
}

// NewSetupCommand creates a new setup command with the given Factory.
// This constructor enables dependency injection for testing.
func NewSetupCommand(f *provider.Factory) *cobra.Command {
	var (
		setupName         string
		setupProviderArgs []string
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Build a sandbox environment",
		Long: `Build a sandbox environment (Docker image or GCP machine image)

GENERAL FLAGS:
  --name <name>              Name for the environment (required)
  --backend <backend>        Backend provider: docker, gcp (default: docker)
  --provider-args <arg>      Extra arguments passed directly to the backend (repeatable)

GCP BACKEND FLAGS:
  --project <project>        GCP project ID (required for GCP)
  --zone <zone>              GCP zone (required for GCP)
  --state-bucket <bucket>    GCS bucket for state persistence (required for GCP)
  --bake-script <path>       Path to custom bake script run during image creation (GCP backend)

EXAMPLES:
  # Docker (default)
  spinner setup --name my-sandbox
  spinner setup --name my-sandbox --provider-args="--build-arg=BASE_IMAGE=node:20-bullseye"

  # Docker with custom Dockerfile
  spinner setup --name my-sandbox --provider-args="-f /path/to/Dockerfile"

  # GCP
  spinner setup --backend gcp --name my-env --project my-proj --zone us-central1-a --state-bucket my-bucket
  spinner setup --backend gcp --name my-env --project my-proj --zone us-central1-a --state-bucket my-bucket \
    --provider-args="--machine-type=e2-standard-4" --provider-args="--boot-disk-size=50GB"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind general flags to Viper
			bindFlags(cmd, flagName)

			// Resolve and validate backend
			backend, err := resolveAndValidateBackend(cmd)
			if err != nil {
				return err
			}

			setupName = viper.GetString(flagName)

			if setupName == "" {
				fmt.Fprintln(os.Stderr, "Error: Missing required flag: --name")
				fmt.Fprintln(os.Stderr, "Usage: spinner setup --name <name> [--backend docker|gcp] [options]")

				return fmt.Errorf("missing required flag: --name")
			}

			p, err := f.Create(backend)
			if err != nil {
				return err
			}

			// Merge provider args: config file provides base args, CLI appends on top.
			providerArgs := mergeProviderArgs(setupProviderArgs)

			return runSetup(context.Background(), p, backend, setupName, providerArgs)
		},
	}

	// General flags
	cmd.Flags().StringVar(&setupName, flagName, "", "Name for the environment (required)")
	cmd.Flags().String(flagBackend, "", "Backend provider: docker, gcp (default: docker)")
	cmd.Flags().StringSliceVar(&setupProviderArgs, flagProviderArgs, []string{}, "Extra arguments passed directly to the backend (repeatable)")

	// GCP backend flags (routing + bake-script only; machine-type, disk-size, service-account
	// are now passed via --provider-args)
	addGCPQueryFlags(cmd)
	cmd.Flags().String(flagBakeScript, "", "Path to custom bake script run during image creation (GCP backend)")

	return cmd
}
