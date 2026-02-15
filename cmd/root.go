package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rickihastings/spinner/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// findConfigFile searches for .spinner.json starting from startDir,
// traversing up to the filesystem root, then checking homeDir as fallback.
// Returns the path to the first file found, or "" if none exists.
func findConfigFile(startDir, homeDir string) string {
	const configFileName = ".spinner.json"

	// Traverse up from startDir to filesystem root
	current := startDir
	for {
		configPath := filepath.Join(current, configFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}

		current = parent
	}

	// Fallback to home directory
	if homeDir != "" {
		homeConfigPath := filepath.Join(homeDir, configFileName)
		if _, err := os.Stat(homeConfigPath); err == nil {
			return homeConfigPath
		}
	}

	return ""
}

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
  spinner spin --image spinner:my-env --repo https://github.com/octocat/Hello-World.git

  # GCP
  spinner setup --backend gcp --name my-env --project my-proj --zone us-central1-a --state-bucket my-bucket
  spinner spin --backend gcp --image my-env --repo https://github.com/octocat/Hello-World.git

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
	// Set environment variable prefix and enable automatic env var reading
	viper.SetEnvPrefix("SPINNER")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Primary config: .spinner.json (traverses from cwd up to root, then checks $HOME)
	cwd, _ := os.Getwd()

	home, _ := os.UserHomeDir()
	if configPath := findConfigFile(cwd, home); configPath != "" {
		viper.SetConfigFile(configPath)

		_ = viper.ReadInConfig() // Ignore error if config can't be read
	}

	// Watch mode defaults
	viper.SetDefault("watch-header", true)

	// Secondary: .env file (not committed, local overrides)
	// Viper only reads one config file, so load .env separately via MergeInConfig.
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	_ = viper.MergeInConfig() // Ignore error if .env doesn't exist
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
