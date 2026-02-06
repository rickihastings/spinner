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
			_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))
			_ = viper.BindPFlag("base-image", cmd.Flags().Lookup("base-image"))
			_ = viper.BindPFlag("dockerfile", cmd.Flags().Lookup("dockerfile"))

			// Bind GCP flags to Viper
			_ = viper.BindPFlag("project", cmd.Flags().Lookup("project"))
			_ = viper.BindPFlag("zone", cmd.Flags().Lookup("zone"))
			_ = viper.BindPFlag("machine-type", cmd.Flags().Lookup("machine-type"))
			_ = viper.BindPFlag("disk-size", cmd.Flags().Lookup("disk-size"))
			_ = viper.BindPFlag("state-bucket", cmd.Flags().Lookup("state-bucket"))

			// Resolve backend (CLI > env > config > default "docker")
			backend = resolveBackend(cmd)

			// Validate cross-backend flags
			if err := validateBackendFlags(cmd, backend); err != nil {
				return err
			}

			// Read values from Viper
			setupName = viper.GetString("name")
			setupBaseImage = viper.GetString("base-image")
			setupDockerfile = viper.GetString("dockerfile")

			if setupName == "" {
				fmt.Fprintln(os.Stderr, "Error: Missing required flag: --name")
				fmt.Fprintln(os.Stderr, "Usage: spinner setup --name <name> [--backend docker|gcp] [options]")

				return fmt.Errorf("missing required flag: --name")
			}

			// Docker-specific validation
			if backend == "docker" {
				if setupBaseImage != "" && setupDockerfile != "" {
					fmt.Fprintln(os.Stderr, "Error: --base-image and --dockerfile are mutually exclusive")
					fmt.Fprintln(os.Stderr, "Please provide only one of these flags")

					return fmt.Errorf("mutually exclusive flags provided")
				}
			}

			// GCP-specific validation
			if backend == "gcp" {
				if err := validateRequiredGCPFlags(cmd); err != nil {
					return err
				}
			}

			// Build options map with all values; the provider picks what it needs
			options := map[string]string{
				"base-image": setupBaseImage,
				"dockerfile": setupDockerfile,
			}

			if backend == "gcp" {
				options["project"] = viper.GetString("project")
				options["zone"] = viper.GetString("zone")
				options["state-bucket"] = viper.GetString("state-bucket")

				mt := viper.GetString("machine-type")
				if mt == "" {
					mt = "e2-standard-2"
				}

				options["machine-type"] = mt

				ds := viper.GetInt("disk-size")
				if ds == 0 {
					ds = 30
				}

				options["disk-size"] = strconv.Itoa(ds)
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
	cmd.Flags().StringVar(&setupName, "name", "", "Name for the environment (required)")
	cmd.Flags().StringVar(&backend, "backend", "", "Backend provider: docker, gcp (default: docker)")

	// Docker backend flags
	cmd.Flags().StringVar(&setupBaseImage, "base-image", "", "Base Docker image (optional, default: ubuntu:22.04)")
	cmd.Flags().StringVar(&setupDockerfile, "dockerfile", "", "Path to custom Dockerfile (optional)")

	// GCP backend flags
	cmd.Flags().StringVar(&project, "project", "", "GCP project ID (GCP backend)")
	cmd.Flags().StringVar(&zone, "zone", "", "GCP zone (GCP backend)")
	cmd.Flags().StringVar(&machineType, "machine-type", "", "VM machine type (GCP backend, default: e2-standard-2)")
	cmd.Flags().IntVar(&diskSize, "disk-size", 0, "Boot disk size in GB (GCP backend, default: 30)")
	cmd.Flags().StringVar(&stateBucket, "state-bucket", "", "GCS bucket for state persistence (GCP backend)")

	return cmd
}
