package cmd

import (
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/setup"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	setupName       string
	setupBaseImage  string
	setupDockerfile string
)

var setupCmd = &cobra.Command{
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
		// Bind flags to viper - this allows environment variables to override flag values
		viper.BindPFlag("name", cmd.Flags().Lookup("name"))
		viper.BindPFlag("base-image", cmd.Flags().Lookup("base-image"))
		viper.BindPFlag("dockerfile", cmd.Flags().Lookup("dockerfile"))

		// Get values from viper (respects env vars and flags)
		setupName = viper.GetString("name")
		setupBaseImage = viper.GetString("base-image")
		setupDockerfile = viper.GetString("dockerfile")

		// Validate required flag
		if setupName == "" {
			fmt.Fprintln(os.Stderr, "Error: Missing required flag: --name")
			fmt.Fprintln(os.Stderr, "Usage: spinner setup --name <name> [--base-image <image> | --dockerfile <path>]")
			return fmt.Errorf("missing required flag: --name")
		}

		// Validate mutually exclusive flags
		if setupBaseImage != "" && setupDockerfile != "" {
			fmt.Fprintln(os.Stderr, "Error: --base-image and --dockerfile are mutually exclusive")
			fmt.Fprintln(os.Stderr, "Please provide only one of these flags")
			return fmt.Errorf("mutually exclusive flags provided")
		}

		// Perform setup using shared logic
		return setup.PerformSetup(setup.Config{
			Name:       setupName,
			BaseImage:  setupBaseImage,
			Dockerfile: setupDockerfile,
		})
	},
}

func init() {
	setupCmd.Flags().StringVar(&setupName, "name", "", "Name for the Docker image (required)")
	setupCmd.Flags().StringVar(&setupBaseImage, "base-image", "", "Base Docker image (optional, default: ubuntu:22.04)")
	setupCmd.Flags().StringVar(&setupDockerfile, "dockerfile", "", "Path to custom Dockerfile (optional)")

	rootCmd.AddCommand(setupCmd)
}
