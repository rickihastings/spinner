package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

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
		setupName       string
		setupBaseImage  string
		setupDockerfile string
		backend         string
		project         string
		zone            string
		machineType     string
		diskSize        int
		stateBucket     string
	)

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

			// Bind GCP flags to Viper
			_ = viper.BindPFlag(flagProject, cmd.Flags().Lookup(flagProject))
			_ = viper.BindPFlag(flagZone, cmd.Flags().Lookup(flagZone))
			_ = viper.BindPFlag(flagMachineType, cmd.Flags().Lookup(flagMachineType))
			_ = viper.BindPFlag(flagDiskSize, cmd.Flags().Lookup(flagDiskSize))
			_ = viper.BindPFlag(flagStateBucket, cmd.Flags().Lookup(flagStateBucket))

			// Resolve backend (CLI > env > config > default "docker")
			backend = resolveBackend(cmd)

			// Validate cross-backend flags
			if err := validateBackendFlags(cmd, backend); err != nil {
				return err
			}

			// Read values from Viper
			setupName = viper.GetString(flagName)
			setupBaseImage = viper.GetString(flagBaseImage)
			setupDockerfile = viper.GetString(flagDockerfile)

			if setupName == "" {
				fmt.Fprintln(os.Stderr, "Error: Missing required flag: --name")
				fmt.Fprintln(os.Stderr, "Usage: spinner setup --name <name> [--backend docker|gcp] [options]")

				return fmt.Errorf("missing required flag: --name")
			}

			// Docker-specific validation
			if backend == provider.BackendDocker {
				if err := validateDockerFlags(cmd); err != nil {
					return err
				}
			}

			// GCP-specific validation
			if backend == provider.BackendGCP {
				if err := validateRequiredGCPFlags(cmd); err != nil {
					return err
				}
			}

			// Build options map with all values; the provider picks what it needs
			options := map[string]string{
				flagBaseImage:  setupBaseImage,
				flagDockerfile: setupDockerfile,
			}

			if backend == provider.BackendGCP {
				options[flagProject] = viper.GetString(flagProject)
				options[flagZone] = viper.GetString(flagZone)
				options[flagStateBucket] = viper.GetString(flagStateBucket)

				mt := viper.GetString(flagMachineType)
				if mt == "" {
					mt = defaultMachineType
				}

				options[flagMachineType] = mt

				ds := viper.GetInt(flagDiskSize)
				if ds == 0 {
					ds = defaultDiskSize
				}

				options[flagDiskSize] = strconv.Itoa(ds)
			}

			// Create provider from factory
			p, err := f.Create(backend)
			if err != nil {
				return err
			}

			return performSetup(context.Background(), p, provider.SetupConfig{
				Name:    setupName,
				Options: options,
			})
		},
	}

	// General flags
	cmd.Flags().StringVar(&setupName, flagName, "", "Name for the environment (required)")
	cmd.Flags().StringVar(&backend, flagBackend, "", "Backend provider: docker, gcp (default: docker)")

	// Docker backend flags
	cmd.Flags().StringVar(&setupBaseImage, flagBaseImage, "", "Base Docker image (optional, default: ubuntu:22.04)")
	cmd.Flags().StringVar(&setupDockerfile, flagDockerfile, "", "Path to custom Dockerfile (optional)")

	// GCP backend flags
	cmd.Flags().StringVar(&project, flagProject, "", "GCP project ID (GCP backend)")
	cmd.Flags().StringVar(&zone, flagZone, "", "GCP zone (GCP backend)")
	cmd.Flags().StringVar(&machineType, flagMachineType, "", "VM machine type (GCP backend, default: e2-standard-2)")
	cmd.Flags().IntVar(&diskSize, flagDiskSize, 0, "Boot disk size in GB (GCP backend, default: 30)")
	cmd.Flags().StringVar(&stateBucket, flagStateBucket, "", "GCS bucket for state persistence (GCP backend)")

	return cmd
}
