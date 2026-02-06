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
	_ = viper.BindPFlag("backend", cmd.Flags().Lookup("backend"))

	backend := viper.GetString("backend")
	if backend == "" {
		backend = "docker"
	}

	return backend
}

// validateBackendFlags ensures no backend-specific CLI flags were passed for
// the wrong backend. Only explicitly-set CLI flags trigger errors; values
// from .spinner.json or env vars are silently ignored.
func validateBackendFlags(cmd *cobra.Command, backend string) error {
	gcpOnlyFlags := []string{"project", "zone", "machine-type", "disk-size", "state-bucket"}
	dockerOnlyFlags := []string{"base-image", "dockerfile"}

	if backend != "gcp" {
		for _, f := range gcpOnlyFlags {
			if cmd.Flags().Lookup(f) != nil && cmd.Flags().Changed(f) {
				return fmt.Errorf("--%s requires --backend gcp", f)
			}
		}
	}

	if backend != "docker" {
		for _, f := range dockerOnlyFlags {
			if cmd.Flags().Lookup(f) != nil && cmd.Flags().Changed(f) {
				return fmt.Errorf("--%s requires --backend docker (or omit --backend)", f)
			}
		}
	}

	return nil
}

// validateRequiredGCPFlags checks that required GCP flags are set (from any
// source: CLI, env, or config file) when backend is "gcp".
func validateRequiredGCPFlags(cmd *cobra.Command) error {
	for _, flag := range []string{"project", "zone", "state-bucket"} {
		if cmd.Flags().Lookup(flag) != nil {
			_ = viper.BindPFlag(flag, cmd.Flags().Lookup(flag))
		}
	}

	if viper.GetString("project") == "" {
		return fmt.Errorf("--project is required for GCP backend")
	}

	if viper.GetString("zone") == "" {
		return fmt.Errorf("--zone is required for GCP backend")
	}

	if viper.GetString("state-bucket") == "" {
		return fmt.Errorf("--state-bucket is required for GCP backend (GCS bucket names are globally unique and must be pre-created)")
	}

	return nil
}
