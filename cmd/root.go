package cmd

import (
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "spinner",
	Short: "CLI tool for running code in isolated sandboxed environments",
	Long: `Spinner - CLI tool for running code in isolated sandboxed environments

USAGE:
  spinner setup --name <name> [--backend docker|gcp] [backend-specific options]
  spinner spin --image <image> --repo <repo> [--backend docker|gcp] [options]

COMMANDS:
  setup    Build a sandbox environment (Docker image or GCP machine image)
  spin     Spin up an instance from a pre-built environment

GENERAL OPTIONS:
  --help                     Show this help message
  --version                  Show version information

EXAMPLES:
  # Docker (default)
  spinner setup --name my-sandbox
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git

  # GCP
  spinner setup --backend gcp --name my-env --project my-proj --zone us-central1-a --state-bucket my-bucket
  spinner spin --backend gcp --image my-env --repo git@github.com:octocat/Hello-World.git

  # Configuration file (.spinner.json) provides defaults for backend-specific flags
  # Precedence: CLI flags > env vars (SPINNER_*) > .spinner.json > defaults

NOTES:
  - Setup: Only Ubuntu/Debian-based images are supported for Docker (requires apt-get)
  - Setup: If using --dockerfile, the custom Dockerfile is built first and used as base
  - Spin: Instance names are deterministic based on image + repo + branch
  - Spin: Running spin with same image/repo/branch reuses the existing instance
  - Spin: Use --recreate to force removal and recreation of existing instance`,
	Version: version.Info(),
}

func init() {
	// Set environment variable prefix
	viper.SetEnvPrefix("SPINNER")
	viper.AutomaticEnv()

	// Primary config: .spinner.json in repo root (committed, team-shared)
	viper.SetConfigName(".spinner")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig() // Ignore error if .spinner.json doesn't exist

	// Secondary: .env file (not committed, local overrides)
	// Viper only reads one config file, so load .env separately via MergeInConfig.
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	_ = viper.MergeInConfig() // Ignore error if .env doesn't exist
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
