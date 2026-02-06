package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// setupCmd is the production setup command using Provider
var setupCmd = NewSetupCommand(docker.NewDockerProvider(docker.NewRealDockerClient()))

func init() {
	rootCmd.AddCommand(setupCmd)
}

// NewSetupCommand creates a new setup command with the given Provider.
// This constructor enables dependency injection for testing.
func NewSetupCommand(p provider.Provider) *cobra.Command {
	var (
		setupName       string
		setupBaseImage  string
		setupDockerfile string
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Build a Docker sandbox image",
		Long: `Build a Docker sandbox image with custom base image or Dockerfile

SETUP OPTIONS:
  --name <name>              Name for the Docker image (required)
  --base-image <image>       Base Docker image (optional, default: ubuntu:22.04)
  --dockerfile <path>        Path to custom Dockerfile (optional, mutually exclusive with --base-image)

EXAMPLES:
  spinner setup --name my-sandbox
  spinner setup --name my-sandbox --base-image ubuntu:22.04
  spinner setup --name node-env --base-image node:20-bullseye
  spinner setup --name custom-env --dockerfile ./Dockerfile.custom`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))
			_ = viper.BindPFlag("base-image", cmd.Flags().Lookup("base-image"))
			_ = viper.BindPFlag("dockerfile", cmd.Flags().Lookup("dockerfile"))

			setupName = viper.GetString("name")
			setupBaseImage = viper.GetString("base-image")
			setupDockerfile = viper.GetString("dockerfile")

			if setupName == "" {
				fmt.Fprintln(os.Stderr, "Error: Missing required flag: --name")
				fmt.Fprintln(os.Stderr, "Usage: spinner setup --name <name> [--base-image <image> | --dockerfile <path>]")

				return fmt.Errorf("missing required flag: --name")
			}

			if setupBaseImage != "" && setupDockerfile != "" {
				fmt.Fprintln(os.Stderr, "Error: --base-image and --dockerfile are mutually exclusive")
				fmt.Fprintln(os.Stderr, "Please provide only one of these flags")

				return fmt.Errorf("mutually exclusive flags provided")
			}

			return performSetup(context.Background(), p, provider.SetupConfig{
				Name:    setupName,
				Options: map[string]string{"base-image": setupBaseImage, "dockerfile": setupDockerfile},
			})
		},
	}

	cmd.Flags().StringVar(&setupName, "name", "", "Name for the Docker image (required)")
	cmd.Flags().StringVar(&setupBaseImage, "base-image", "", "Base Docker image (optional, default: ubuntu:22.04)")
	cmd.Flags().StringVar(&setupDockerfile, "dockerfile", "", "Path to custom Dockerfile (optional)")

	return cmd
}
