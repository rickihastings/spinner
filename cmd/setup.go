package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/prerequisites"
	"github.com/spf13/cobra"
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

		// Check prerequisites
		fmt.Println("Checking prerequisites...")
		if err := prerequisites.CheckPrerequisites(); err != nil {
			fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
			return err
		}

		// Validate Dockerfile path if provided
		if setupDockerfile != "" {
			if _, err := os.Stat(setupDockerfile); os.IsNotExist(err) {
				errMsg := fmt.Sprintf("Dockerfile not found at path: %s", setupDockerfile)
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", errMsg)
				return errors.New(errMsg)
			}
		}

		// Build the image
		fmt.Printf("✓ Prerequisites checked\n")
		fmt.Printf("Building Docker image: spinner:%s\n", setupName)

		buildConfig := docker.BuildConfig{
			Name:       setupName,
			BaseImage:  setupBaseImage,
			Dockerfile: setupDockerfile,
		}

		if err := docker.BuildImage(buildConfig); err != nil {
			fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
			return err
		}

		fmt.Printf("✓ Docker image built successfully: spinner:%s\n", setupName)
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&setupName, "name", "", "Name for the Docker image (required)")
	setupCmd.Flags().StringVar(&setupBaseImage, "base-image", "", "Base Docker image (optional, default: ubuntu:22.04)")
	setupCmd.Flags().StringVar(&setupDockerfile, "dockerfile", "", "Path to custom Dockerfile (optional)")

	rootCmd.AddCommand(setupCmd)
}
