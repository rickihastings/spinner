package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flag name constants shared across commands.
const (
	flagBackend        = "backend"
	flagName           = "name"
	flagImage          = "image"
	flagRepo           = "repo"
	flagPrompt         = "prompt"
	flagBranch         = "branch"
	flagMaxIterations  = "max-iterations"
	flagRecreate       = "recreate"
	flagSetup          = "setup"
	flagWatch          = "watch"
	flagBaseImage      = "base-image"
	flagDockerfile     = "dockerfile"
	flagProject        = "project"
	flagZone           = "zone"
	flagMachineType    = "machine-type"
	flagDiskSize       = "disk-size"
	flagStateBucket    = "state-bucket"
	flagBakeScript     = "bake-script"
	flagServiceAccount = "service-account"
	flagEnv            = "env"
	flagEnvFile        = "env-file"
	flagModel          = "model"
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

// resolveAndValidateBackend resolves the backend from flags/config/env and runs
// all backend-specific validation in one call. This replaces the repeated sequence
// of bindGCPFlags + resolveBackend + validateBackendFlags + validateDockerFlags +
// validateRequiredGCPFlags that was duplicated across setup, spin, and watch.
func resolveAndValidateBackend(cmd *cobra.Command) (string, error) {
	bindGCPFlags(cmd)

	_ = viper.BindPFlag(flagBackend, cmd.Flags().Lookup(flagBackend))

	backend := viper.GetString(flagBackend)
	if backend == "" {
		backend = provider.BackendDocker
	}

	if err := validateBackendFlags(cmd, backend); err != nil {
		return "", err
	}

	if backend == provider.BackendDocker {
		if err := validateDockerFlags(cmd); err != nil {
			return "", err
		}
	}

	if backend == provider.BackendGCP {
		if err := validateRequiredGCPFlags(cmd); err != nil {
			return "", err
		}

		if err := validateBakeScriptFlag(cmd); err != nil {
			return "", err
		}
	}

	return backend, nil
}

// buildSetupOptions builds the merged options map for environment setup,
// combining Docker and GCP options based on the active backend.
func buildSetupOptions(backend string) map[string]string {
	options := dockerSetupOptions()

	if backend == provider.BackendGCP {
		for k, v := range gcpOptionsFromViper() {
			options[k] = v
		}
	}

	return options
}

// runSetup executes the full environment setup workflow for any backend.
// Used by both the setup command and spin --setup.
func runSetup(ctx context.Context, p provider.Provider, backend, name string) error {
	return performSetup(ctx, p, provider.SetupConfig{
		Name:    name,
		Options: buildSetupOptions(backend),
	})
}

// validateBackendFlags ensures no backend-specific CLI flags were passed for
// the wrong backend. Only explicitly-set CLI flags trigger errors; values
// from .spinner.json or env vars are silently ignored.
func validateBackendFlags(cmd *cobra.Command, backend string) error {
	gcpOnlyFlags := []string{flagProject, flagZone, flagMachineType, flagDiskSize, flagStateBucket, flagBakeScript, flagServiceAccount}
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

// validateBakeScriptFlag checks that --bake-script is valid when provided:
// - File must exist at the given path
// - On spin command, --bake-script requires --setup
func validateBakeScriptFlag(cmd *cobra.Command) error {
	bakeScriptFlag := cmd.Flags().Lookup(flagBakeScript)
	if bakeScriptFlag == nil {
		return nil
	}

	_ = viper.BindPFlag(flagBakeScript, bakeScriptFlag)

	bakeScript := viper.GetString(flagBakeScript)
	if bakeScript == "" {
		return nil
	}

	// If the command has a --setup flag (spin command), --bake-script requires --setup
	if setupFlag := cmd.Flags().Lookup(flagSetup); setupFlag != nil {
		setup, _ := cmd.Flags().GetBool(flagSetup)
		if !setup {
			return fmt.Errorf("--%s requires --%s flag", flagBakeScript, flagSetup)
		}
	}

	// Validate file exists
	if _, err := os.Stat(bakeScript); err != nil {
		return fmt.Errorf("bake script not found: %s", bakeScript)
	}

	return nil
}

// gcpFlags is the list of all GCP-specific flag names.
var gcpFlags = []string{flagProject, flagZone, flagMachineType, flagDiskSize, flagStateBucket, flagBakeScript, flagServiceAccount}

// bindGCPFlags binds all GCP-specific flags on cmd to Viper.
// Skips flags that don't exist on the command (e.g. watch doesn't have machine-type).
func bindGCPFlags(cmd *cobra.Command) {
	for _, f := range gcpFlags {
		if fl := cmd.Flags().Lookup(f); fl != nil {
			_ = viper.BindPFlag(f, fl)
		}
	}
}

// gcpOptionsFromViper reads GCP flags from Viper and returns an options map
// with defaults applied for machine-type and disk-size.
func gcpOptionsFromViper() map[string]string {
	mt := viper.GetString(flagMachineType)
	if mt == "" {
		mt = defaultMachineType
	}

	ds := viper.GetInt(flagDiskSize)
	if ds == 0 {
		ds = defaultDiskSize
	}

	return map[string]string{
		flagProject:        viper.GetString(flagProject),
		flagZone:           viper.GetString(flagZone),
		flagStateBucket:    viper.GetString(flagStateBucket),
		flagMachineType:    mt,
		flagDiskSize:       strconv.Itoa(ds),
		flagBakeScript:     viper.GetString(flagBakeScript),
		flagServiceAccount: viper.GetString(flagServiceAccount),
	}
}

// dockerSetupOptions returns the Docker-specific options for a setup config.
func dockerSetupOptions() map[string]string {
	return map[string]string{
		flagBaseImage:  viper.GetString(flagBaseImage),
		flagDockerfile: viper.GetString(flagDockerfile),
	}
}
