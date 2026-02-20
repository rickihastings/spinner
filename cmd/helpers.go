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
	flagDockerfile    = "dockerfile"
	flagProject       = "project"
	flagZone          = "zone"
	flagStateBucket   = "state-bucket"
	flagBakeScript    = "bake-script"
	flagEnv           = "env"
	flagEnvFile       = "env-file"
	flagModel         = "model"
	flagProviderArgs  = "provider-args"
	flagSecret        = "secret"

	// Config file keys for command-specific provider args.
	// Using separate keys prevents build flags (e.g. -f ./Dockerfile) from
	// leaking into docker run, and run flags from leaking into docker build.
	configSetupProviderArgs = "setup-provider-args"
	configSpinProviderArgs  = "spin-provider-args"
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
// of bindGCPFlags + resolveBackend + validateBackendFlags +
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

// buildSetupOptions builds the options map for environment setup.
// For GCP, this includes routing options (project, zone, state-bucket, bake-script).
// For Docker, this includes the optional custom Dockerfile path.
func buildSetupOptions(backend string) map[string]string {
	if backend == provider.BackendGCP {
		return gcpRoutingOptions()
	}

	return map[string]string{
		flagDockerfile: viper.GetString(flagDockerfile),
	}
}

// runSetup executes the full environment setup workflow for any backend.
// Used by both the setup command and spin --setup.
// secretBlob and secretKey carry the encrypted secret material for GCP bake VMs;
// pass nil for both when secrets are not needed (e.g. spin --setup, Docker).
func runSetup(ctx context.Context, p provider.Provider, backend, name string, secretBlob, secretKey []byte, providerArgs []string) error {
	return performSetup(ctx, p, provider.SetupConfig{
		Name:         name,
		Options:      buildSetupOptions(backend),
		SecretBlob:   secretBlob,
		SecretKey:    secretKey,
		ProviderArgs: providerArgs,
	})
}

// validateBackendFlags ensures no backend-specific CLI flags were passed for
// the wrong backend. Only explicitly-set CLI flags trigger errors; values
// from .spinner.json or env vars are silently ignored.
func validateBackendFlags(cmd *cobra.Command, backend string) error {
	gcpOnlyFlags := []string{flagProject, flagZone, flagStateBucket, flagBakeScript}

	if backend != provider.BackendGCP {
		for _, f := range gcpOnlyFlags {
			if cmd.Flags().Lookup(f) != nil && cmd.Flags().Changed(f) {
				return fmt.Errorf("--%s requires --backend %s", f, provider.BackendGCP)
			}
		}
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
var gcpFlags = []string{flagProject, flagZone, flagStateBucket, flagBakeScript}

// bindGCPFlags binds all GCP-specific flags on cmd to Viper.
// Skips flags that don't exist on the command (e.g. watch doesn't have machine-type).
func bindGCPFlags(cmd *cobra.Command) {
	for _, f := range gcpFlags {
		if fl := cmd.Flags().Lookup(f); fl != nil {
			_ = viper.BindPFlag(f, fl)
		}
	}
}

// gcpRoutingOptions returns the GCP routing options (project, zone, state-bucket,
// bake-script) from Viper. These are the flags that remain first-class; tuning
// flags like machine-type, disk-size, and service-account are now passed via
// --provider-args.
func gcpRoutingOptions() map[string]string {
	return map[string]string{
		flagProject:     viper.GetString(flagProject),
		flagZone:        viper.GetString(flagZone),
		flagStateBucket: viper.GetString(flagStateBucket),
		flagBakeScript:  viper.GetString(flagBakeScript),
	}
}

// addGCPQueryFlags registers the minimal GCP flags needed for querying instances.
// Used by watch and destroy which only need project/zone/bucket to locate instances.
func addGCPQueryFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagProject, "", "GCP project ID (GCP backend)")
	cmd.Flags().String(flagZone, "", "GCP zone (GCP backend)")
	cmd.Flags().String(flagStateBucket, "", "GCS bucket for state persistence (GCP backend)")
}

// bindFlags binds the named flags on cmd to Viper.
func bindFlags(cmd *cobra.Command, flags ...string) {
	for _, f := range flags {
		_ = viper.BindPFlag(f, cmd.Flags().Lookup(f))
	}
}

// mergeProviderArgs combines provider args from .spinner.json (via Viper)
// with CLI-provided args. Config file args form the base; CLI args are appended.
// If the same flag appears in both, the backend tool uses its own last-wins semantics.
//
// configKey is the command-specific .spinner.json key to read (e.g.
// configSetupProviderArgs or configSpinProviderArgs). Using separate keys prevents
// build-only flags (e.g. -f ./Dockerfile) from leaking into docker run, and vice versa.
//
// Note: --provider-args is NOT bound to Viper via BindPFlag so that both sources
// can be read independently. Viper reads config file values; cliArgs comes from Cobra.
func mergeProviderArgs(configKey string, cliArgs []string) []string {
	configArgs := viper.GetStringSlice(configKey)
	if len(cliArgs) == 0 {
		return configArgs
	}

	if len(configArgs) == 0 {
		return cliArgs
	}

	merged := make([]string, 0, len(configArgs)+len(cliArgs))
	merged = append(merged, configArgs...)
	merged = append(merged, cliArgs...)

	return merged
}
