package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "spinner",
	Short: "CLI tool for running code in isolated Docker containers",
	Long: `Spinner - CLI tool for running code in isolated Docker containers

USAGE:
  spinner setup --name <name> [--base-image <image> | --dockerfile <path>]
  spinner spin --image <image> --repo <repo> [--prompt <prompt> --branch <branch> [--max-iterations <num>]]

COMMANDS:
  setup    Build a Docker sandbox image with custom base image or Dockerfile
  spin     Spin up a development container from a pre-built image

GENERAL OPTIONS:
  --help                     Show this help message
  --version                  Show version information

EXAMPLES:
  spinner setup --name my-sandbox
  spinner setup --name my-sandbox --base-image ubuntu:22.04
  spinner setup --name node-env --base-image node:20-bullseye
  spinner setup --name custom-env --dockerfile ./Dockerfile.custom
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git --prompt "Implement feature X"
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git --prompt "Implement feature X" --branch feature/x

NOTES:
  - Setup: Only Ubuntu/Debian-based images are supported (requires apt-get)
  - Setup: The CLI ensures git and claude-code are installed in the final image
  - Setup: If using --dockerfile, the custom Dockerfile is built first and used as base
  - Spin: SSH agent must be running on host system
  - Spin: Container names are deterministic based on image + repo + branch
  - Spin: Running spin with same image/repo/branch reuses the existing container
  - Spin: Use --recreate to force removal and recreation of existing container
  - Spin: Containers are persistent and must be manually stopped/removed`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
