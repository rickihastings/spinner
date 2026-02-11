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
	var setupName string

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Build a sandbox environment",
		Long: `Build a sandbox environment (Docker image or GCP machine image)

GENERAL FLAGS:
  --name <name>              Name for the environment (required)
  --backend <backend>        Backend provider: docker, gcp (default: docker)

DOCKER BACKEND FLAGS:
  --base-image <image>       Base Docker image (optional, default: ubuntu:22.04)
  --dockerfile <path>        Path to custom Dockerfile (optional, mutually exclusive with --base-image)

GCP BACKEND FLAGS:
  --project <project>        GCP project ID (required for GCP)
  --zone <zone>              GCP zone (required for GCP)
  --machine-type <type>      VM machine type (default: e2-standard-2)
  --disk-size <gb>           Boot disk size in GB (default: 30)
  --state-bucket <bucket>    GCS bucket for state persistence (required for GCP)
  --bake-script <path>       Path to custom bake script run during image creation (GCP backend)

EXAMPLES:
  # Docker (default)
  spinner setup --name my-sandbox
  spinner setup --name my-sandbox --base-image node:20-bullseye

  # GCP
  spinner setup --backend gcp --name my-env --project my-proj --zone us-central1-a --state-bucket my-bucket`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind general flags to Viper
			_ = viper.BindPFlag(flagName, cmd.Flags().Lookup(flagName))
			_ = viper.BindPFlag(flagBaseImage, cmd.Flags().Lookup(flagBaseImage))
			_ = viper.BindPFlag(flagDockerfile, cmd.Flags().Lookup(flagDockerfile))

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

			return runSetup(context.Background(), p, backend, setupName)
		},
	}

	// General flags
	cmd.Flags().StringVar(&setupName, flagName, "", "Name for the environment (required)")
	cmd.Flags().String(flagBackend, "", "Backend provider: docker, gcp (default: docker)")

	// Docker backend flags
	cmd.Flags().String(flagBaseImage, "", "Base Docker image (optional, default: ubuntu:22.04)")
	cmd.Flags().String(flagDockerfile, "", "Path to custom Dockerfile (optional)")

	// GCP backend flags
	cmd.Flags().String(flagProject, "", "GCP project ID (GCP backend)")
	cmd.Flags().String(flagZone, "", "GCP zone (GCP backend)")
	cmd.Flags().String(flagMachineType, "", "VM machine type (GCP backend, default: e2-standard-2)")
	cmd.Flags().Int(flagDiskSize, 0, "Boot disk size in GB (GCP backend, default: 30)")
	cmd.Flags().String(flagStateBucket, "", "GCS bucket for state persistence (GCP backend)")
	cmd.Flags().String(flagBakeScript, "", "Path to custom bake script run during image creation (GCP backend)")
	cmd.Flags().String(flagServiceAccount, "", "GCP service account email (GCP backend)")

	return cmd
}
