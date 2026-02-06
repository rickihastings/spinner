package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flag name constants shared across commands.
const (
	flagBackend       = "backend"
	flagName          = "name"
	flagImage         = "image"
	flagRepo          = "repo"
	flagPrompt        = "prompt"
	flagBranch        = "branch"
	flagMaxIterations = "max-iterations"
	flagRecreate      = "recreate"
	flagSetup         = "setup"
	flagWatch         = "watch"
	flagBaseImage     = "base-image"
	flagDockerfile    = "dockerfile"
	flagProject       = "project"
	flagZone          = "zone"
	flagMachineType   = "machine-type"
	flagDiskSize      = "disk-size"
	flagStateBucket   = "state-bucket"
)

// GCP default values.
const (
	defaultMachineType = "e2-standard-2"
	defaultDiskSize    = 30
)

// performSetup runs the shared setup workflow: environment provisioning.
// Called by both the setup and spin --setup paths.
func performSetup(ctx context.Context, p provider.Provider, config provider.SetupConfig) error {
	fmt.Printf("Provisioning environment: %s\n", config.Name)

	if err := p.Setup(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
		return err
	}

	fmt.Printf("✓ Environment provisioned: %s\n", config.Name)

	return nil
}

// isValidGitURL checks whether the given string is a valid git URL.
func isValidGitURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "git@")
}

// resolveBackend reads the --backend value from Viper (CLI > env > config > default).
func resolveBackend(cmd *cobra.Command) string {
	_ = viper.BindPFlag(flagBackend, cmd.Flags().Lookup(flagBackend))

	backend := viper.GetString(flagBackend)
	if backend == "" {
		backend = provider.BackendDocker
	}

	return backend
}

// validateBackendFlags ensures no backend-specific CLI flags were passed for
// the wrong backend. Only explicitly-set CLI flags trigger errors; values
// from .spinner.json or env vars are silently ignored.
func validateBackendFlags(cmd *cobra.Command, backend string) error {
	gcpOnlyFlags := []string{flagProject, flagZone, flagMachineType, flagDiskSize, flagStateBucket}
	dockerOnlyFlags := []string{flagBaseImage, flagDockerfile}

	if backend != provider.BackendGCP {
		for _, f := range gcpOnlyFlags {
			if cmd.Flags().Lookup(f) != nil && cmd.Flags().Changed(f) {
				return fmt.Errorf("--%s requires --backend %s", f, provider.BackendGCP)
			}
		}
	}

	if backend != provider.BackendDocker {
		for _, f := range dockerOnlyFlags {
			if cmd.Flags().Lookup(f) != nil && cmd.Flags().Changed(f) {
				return fmt.Errorf("--%s requires --backend %s (or omit --backend)", f, provider.BackendDocker)
			}
		}
	}

	return nil
}

// validateDockerFlags checks Docker-specific flag constraints.
// When the command has a --setup flag (e.g. spin), --base-image and --dockerfile
// require --setup. In all cases, --base-image and --dockerfile are mutually exclusive.
func validateDockerFlags(cmd *cobra.Command) error {
	baseImage := viper.GetString(flagBaseImage)
	dockerfile := viper.GetString(flagDockerfile)

	// If the command has a --setup flag, build flags require it
	if setupFlag := cmd.Flags().Lookup(flagSetup); setupFlag != nil {
		setup, _ := cmd.Flags().GetBool(flagSetup)

		if !setup && baseImage != "" {
			fmt.Fprintf(os.Stderr, "Error: --%s requires --%s flag\n", flagBaseImage, flagSetup)
			return fmt.Errorf("--%s requires --%s flag", flagBaseImage, flagSetup)
		}

		if !setup && dockerfile != "" {
			fmt.Fprintf(os.Stderr, "Error: --%s requires --%s flag\n", flagDockerfile, flagSetup)
			return fmt.Errorf("--%s requires --%s flag", flagDockerfile, flagSetup)
		}
	}

	if baseImage != "" && dockerfile != "" {
		fmt.Fprintln(os.Stderr, "Error: --base-image and --dockerfile are mutually exclusive")
		fmt.Fprintln(os.Stderr, "Please provide only one of these flags")

		return fmt.Errorf("mutually exclusive flags provided")
	}

	return nil
}

// validateRequiredGCPFlags checks that required GCP flags are set (from any
// source: CLI, env, or config file) when backend is "gcp".
func validateRequiredGCPFlags(cmd *cobra.Command) error {
	for _, flag := range []string{flagProject, flagZone, flagStateBucket} {
		if cmd.Flags().Lookup(flag) != nil {
			_ = viper.BindPFlag(flag, cmd.Flags().Lookup(flag))
		}
	}

	if viper.GetString(flagProject) == "" {
		return fmt.Errorf("--%s is required for GCP backend", flagProject)
	}

	if viper.GetString(flagZone) == "" {
		return fmt.Errorf("--%s is required for GCP backend", flagZone)
	}

	if viper.GetString(flagStateBucket) == "" {
		return fmt.Errorf("--%s is required for GCP backend (GCS bucket names are globally unique and must be pre-created)", flagStateBucket)
	}

	return nil
}
