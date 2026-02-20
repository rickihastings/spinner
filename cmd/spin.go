package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rickihastings/spinner/internal/secret"
	"github.com/rickihastings/spinner/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// spinStoreFactory creates the secret store used by spin.
// Override in tests to inject a mock store.
var spinStoreFactory = func() secret.Store {
	return secret.NewEncryptedFileStore("", passphraseFromEnvOrPrompt)
}

// spinCmd is the production spin command using the default provider factory.
var spinCmd = NewSpinCommand(defaultFactory)

func init() {
	rootCmd.AddCommand(spinCmd)
}

// NewSpinCommand creates a new spin command with the given Factory.
// This constructor enables dependency injection for testing.
func NewSpinCommand(f *provider.Factory) *cobra.Command {
	var (
		spinImage         string
		spinRepo          string
		spinPrompt        string
		spinBranch        string
		spinMaxIterations string
		spinModel         string
		spinRecreate      bool
		spinSetup         bool
		spinWatch         bool
		spinEnvVars       []string
		spinEnvFile       string
		spinSecrets       []string
		spinProviderArgs  []string
		spinDockerfile    string
	)

	cmd := &cobra.Command{
		Use:   "spin",
		Short: "Spin up an instance from a pre-built environment",
		Long: `Spin up an instance from a pre-built environment

GENERAL FLAGS:
  --image <image>            Environment image to use (required)
  --repo <repo>              Git repository URL (required)
  --prompt <prompt>          Task prompt for autonomous execution (optional)
  --branch <branch>          Git branch to checkout (optional)
  --max-iterations <num>     Maximum iterations (optional, default: 100)
  --model <model>            Claude model to use (optional, e.g. claude-sonnet-4-5-20250929)
  --recreate                 Force recreation of existing instance (optional)
  --watch                    Enter watch mode after instance is ready (optional)
  --backend <backend>        Backend provider: docker, gcp (default: docker)
  --secret <KEY>             Reference a secret from the encrypted store (repeatable)
  --env <KEY=VALUE>          Custom environment variables (repeatable)
  --env-file <path>          Path to env file to pass to instance (optional)
  --provider-args <arg>      Extra arguments passed directly to the backend (repeatable)

SETUP OPTIONS (use with --setup flag):
  --setup                    Build/rebuild the environment before spinning (optional)
  --dockerfile <path>        Custom Dockerfile to use as the base image (Docker backend, used with --setup)

GCP BACKEND FLAGS:
  --project <project>        GCP project ID (required for GCP)
  --zone <zone>              GCP zone (required for GCP)
  --state-bucket <bucket>    GCS bucket for state persistence (required for GCP)
  --bake-script <path>       Path to custom bake script run during image creation (GCP backend, requires --setup)

EXAMPLES:
  # Docker (default)
  spinner spin --image spinner:my-env --repo https://github.com/octocat/Hello-World.git
  spinner spin --setup --image my-env --repo https://github.com/octocat/Hello-World.git --prompt "Fix the bug"

  # Docker with provider args
  spinner spin --image spinner:my-env --repo https://github.com/octocat/Hello-World.git \
    --provider-args="-v /data:/data" --provider-args="--network=host"

  # GCP
  spinner spin --backend gcp --image my-env --repo https://github.com/octocat/Hello-World.git \
    --project my-proj --zone us-central1-a --state-bucket my-bucket --prompt "Fix the bug"

  # GCP with provider args
  spinner spin --backend gcp --image my-env --repo https://github.com/octocat/Hello-World.git \
    --project my-proj --zone us-central1-a --state-bucket my-bucket \
    --provider-args="--machine-type=e2-standard-4" --provider-args="--boot-disk-size=50GB"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind general flags to Viper
			bindFlags(cmd,
				flagImage, flagRepo, flagPrompt, flagBranch,
				flagMaxIterations, flagModel, flagRecreate,
				flagSetup, flagWatch, flagDockerfile,
			)

			// Resolve and validate backend
			backend, err := resolveAndValidateBackend(cmd)
			if err != nil {
				return err
			}

			// Read values from Viper
			spinImage = viper.GetString(flagImage)
			spinRepo = viper.GetString(flagRepo)
			spinPrompt = viper.GetString(flagPrompt)
			spinBranch = viper.GetString(flagBranch)
			spinMaxIterations = viper.GetString(flagMaxIterations)
			spinModel = viper.GetString(flagModel)
			spinRecreate = viper.GetBool(flagRecreate)
			spinSetup = viper.GetBool(flagSetup)
			spinWatch = viper.GetBool(flagWatch)
			spinDockerfile = viper.GetString(flagDockerfile)

			// Parse and validate custom env vars
			envVars, err := parseAndValidateEnvVars(spinEnvVars)
			if err != nil {
				return err
			}

			// Validate env file exists if provided
			if spinEnvFile != "" {
				if _, err := os.Stat(spinEnvFile); err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("--env-file: file does not exist: %s", spinEnvFile)
					}

					return fmt.Errorf("--env-file: cannot read file: %w", err)
				}
			}

			if spinImage == "" {
				return fmt.Errorf("--image flag is required")
			}

			if spinRepo == "" {
				return fmt.Errorf("--repo flag is required")
			}

			// Create provider from factory
			p, err := f.Create(backend)
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Merge provider args: config file provides base args, CLI appends on top.
			providerArgs := mergeProviderArgs(configSpinProviderArgs, spinProviderArgs)

			// If --setup is provided, provision the environment first
			if spinSetup {
				setupName := strings.TrimPrefix(spinImage, "spinner:")

				if err := runSetup(ctx, p, backend, setupName, nil, nil, providerArgs); err != nil {
					return err
				}

				if backend == provider.BackendDocker {
					spinImage = "spinner:" + setupName
				}
			}

			// Validate prerequisites
			fmt.Println("Validating prerequisites...")

			if !isValidGitURL(spinRepo) {
				fmt.Fprintln(os.Stderr, "✗ Error: Repository must be a valid git URL (https://, http://, or git@)")
				return fmt.Errorf("repository must be a valid git URL (https://, http://, or git@)")
			}

			// Resolve all secrets from the encrypted store (no env var fallback)
			store := spinStoreFactory()

			resolved, err := secret.Resolve(store, spinSecrets)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
				return err
			}

			// Encrypt resolved secrets into a per-session blob with a random key
			key, blob, err := secret.EncryptBlobWithKey(resolved)
			if err != nil {
				return fmt.Errorf("encrypting secrets: %w", err)
			}

			fmt.Println("✓ Prerequisites validated")

			createOptions := map[string]string{flagImage: spinImage}

			if backend == provider.BackendGCP {
				for k, v := range gcpRoutingOptions() {
					createOptions[k] = v
				}
			}

			// Convert SSH URLs to HTTPS so all backends use PAT auth via GITHUB_TOKEN.
			spinRepo = util.ConvertSshToHttps(spinRepo)

			createConfig := provider.CreateConfig{
				Repo:          spinRepo,
				Prompt:        spinPrompt,
				Branch:        spinBranch,
				MaxIterations: spinMaxIterations,
				Model:         spinModel,
				Options:       createOptions,
				EnvVars:       envVars,
				EnvFile:       spinEnvFile,
				SecretBlob:    blob,
				SecretKey:     key,
				ProviderArgs:  providerArgs,
			}

			name := p.InstanceName(createConfig)

			// Query current instance state
			status, err := p.Status(ctx, name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
				return err
			}

			// --recreate: destroy the existing instance so we fall through to create
			if spinRecreate && status != provider.InstanceStatusNone {
				if err := p.Remove(ctx, name); err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
					return err
				}

				status = provider.InstanceStatusNone
			}

			var instance *provider.Instance

			switch status {
			case provider.InstanceStatusNone:
				fmt.Printf("Creating instance: %s\n", name)
				fmt.Println("Cloning repository...")

				instance, err = p.Create(ctx, createConfig)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
					return err
				}

				fmt.Printf("✓ Instance created successfully: %s\n", instance.Name)
			case provider.InstanceStatusRunning:
				instance = &provider.Instance{Name: name, Status: provider.InstanceStatusRunning}

				fmt.Printf("✓ Reusing running instance: %s\n", name)
				fmt.Println()
				fmt.Println("Note: Reusing existing instance. Use --recreate flag to force recreation.")
			case provider.InstanceStatusStopped:
				instance, err = p.Start(ctx, name, createConfig)
				if err != nil {
					// Instance may have been deleted between Status() and Start() (TOCTOU race).
					// Re-check: if gone, create a fresh one instead of failing.
					retryStatus, statusErr := p.Status(ctx, name)
					if statusErr == nil && retryStatus == provider.InstanceStatusNone {
						fmt.Println("Instance was removed, creating fresh instance...")

						instance, err = p.Create(ctx, createConfig)
						if err != nil {
							fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
							return err
						}

						fmt.Printf("✓ Instance created successfully: %s\n", instance.Name)
					} else {
						fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
						return err
					}
				} else {
					fmt.Printf("✓ Instance restarted: %s\n", instance.Name)
					fmt.Println()
					fmt.Println("Note: Reusing existing instance. Use --recreate flag to force recreation.")
				}
			}

			// Display backend-specific management commands
			fmt.Println()

			if backend == provider.BackendGCP {
				gcpProject := viper.GetString(flagProject)
				gcpZone := viper.GetString(flagZone)

				fmt.Printf("To access: gcloud compute ssh %s --project %s --zone %s\n", instance.Name, gcpProject, gcpZone)
				fmt.Printf("To stop:   gcloud compute instances stop %s --project %s --zone %s\n", instance.Name, gcpProject, gcpZone)
				fmt.Printf("To destroy: spinner destroy %s\n", instance.Name)
			} else {
				fmt.Printf("To access: docker exec -it %s bash\n", instance.Name)
				fmt.Printf("To stop: docker stop %s\n", instance.Name)
				fmt.Printf("To destroy: spinner destroy %s\n", instance.Name)
			}

			// Enter watch mode if --watch flag is set
			if spinWatch {
				fmt.Println()
				fmt.Println("Entering watch mode...")

				return performWatch(ctx, p, instance.Name)
			}

			return nil
		},
	}

	// General flags
	cmd.Flags().StringVar(&spinImage, flagImage, "", "Environment image to use (required)")
	cmd.Flags().StringVar(&spinRepo, flagRepo, "", "Git repository URL (required)")
	cmd.Flags().StringVar(&spinPrompt, flagPrompt, "", "Task prompt for autonomous execution (optional)")
	cmd.Flags().StringVar(&spinBranch, flagBranch, "", "Git branch to checkout (optional)")
	cmd.Flags().StringVar(&spinMaxIterations, flagMaxIterations, "", "Maximum iterations (optional, default: 100)")
	cmd.Flags().StringVar(&spinModel, flagModel, "", "Claude model to use (optional, e.g. claude-sonnet-4-5-20250929)")
	cmd.Flags().BoolVar(&spinRecreate, flagRecreate, false, "Force recreation of existing instance (optional)")
	cmd.Flags().BoolVar(&spinSetup, flagSetup, false, "Build/rebuild the environment before spinning (optional)")
	cmd.Flags().String(flagBackend, "", "Backend provider: docker, gcp (default: docker)")
	cmd.Flags().BoolVar(&spinWatch, flagWatch, false, "Enter watch mode after instance is ready (optional)")
	cmd.Flags().StringSliceVar(&spinSecrets, flagSecret, []string{}, "Reference a secret from the encrypted store (repeatable)")
	cmd.Flags().StringSliceVar(&spinEnvVars, flagEnv, []string{}, "Custom environment variables (KEY=VALUE, repeatable)")
	cmd.Flags().StringVar(&spinEnvFile, flagEnvFile, "", "Path to env file to pass to instance (optional)")
	cmd.Flags().StringSliceVar(&spinProviderArgs, flagProviderArgs, []string{}, "Extra arguments passed directly to the backend (repeatable)")

	// Docker backend flags (only valid with --setup)
	cmd.Flags().StringVar(&spinDockerfile, flagDockerfile, "", "Path to custom Dockerfile to use as the base image (Docker backend, used with --setup)")

	// GCP backend flags (routing + bake-script only; machine-type, disk-size, service-account
	// are now passed via --provider-args)
	addGCPQueryFlags(cmd)
	cmd.Flags().String(flagBakeScript, "", "Path to custom bake script run during image creation (GCP backend)")

	return cmd
}

// parseAndValidateEnvVars parses a list of KEY=VALUE strings into a map
// and validates the format and reserved variable names.
func parseAndValidateEnvVars(envVars []string) (map[string]string, error) {
	result := make(map[string]string)

	// Reserved variables that cannot be overridden
	reserved := map[string]bool{
		"GITHUB_TOKEN":              true,
		"CLAUDE_CODE_OAUTH_TOKEN":   true,
		"REPO_URL":                  true,
		"PROMPT":                    true,
		"BRANCH":                    true,
		"MAX_ITERATIONS":            true,
		"LOG_DIR":                   true,
		"STATE_DIR":                 true,
		"SPINNER_LOG_BUCKET":        true,
		"SPINNER_STATE_BUCKET":      true,
		"SPINNER_INSTANCE_NAME":     true,
		"ANTHROPIC_MODEL":           true,
		"SPINNER_SECRET_PASSPHRASE": true,
		"SPINNER_SECRET_KEY":        true,
	}

	for _, env := range envVars {
		// Split on first '=' only
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("--env: invalid format %q, expected KEY=VALUE", env)
		}

		key := parts[0]
		value := parts[1]

		// Validate key is not empty
		if key == "" {
			return nil, fmt.Errorf("--env: key must not be empty")
		}

		// Validate key is not reserved
		if reserved[key] {
			return nil, fmt.Errorf("--env: cannot override reserved variable %q", key)
		}

		result[key] = value
	}

	return result, nil
}
