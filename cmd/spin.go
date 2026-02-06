package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rickihastings/spinner/internal/prerequisites"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
		spinRecreate      bool
		spinSetup         bool
		spinBaseImage     string
		spinDockerfile    string
		spinWatch         bool
		backend           string
		project           string
		zone              string
		machineType       string
		diskSize          int
		stateBucket       string
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
  --recreate                 Force recreation of existing instance (optional)
  --watch                    Enter watch mode after instance is ready (optional)
  --backend <backend>        Backend provider: docker, gcp (default: docker)

SETUP OPTIONS (use with --setup flag):
  --setup                    Build/rebuild the environment before spinning (optional)
  --base-image <image>       Base Docker image (Docker backend, requires --setup)
  --dockerfile <path>        Path to custom Dockerfile (Docker backend, requires --setup)

GCP BACKEND FLAGS:
  --project <project>        GCP project ID (required for GCP)
  --zone <zone>              GCP zone (required for GCP)
  --machine-type <type>      VM machine type (default: e2-standard-2)
  --disk-size <gb>           Boot disk size in GB (default: 30)
  --state-bucket <bucket>    GCS bucket for state persistence (required for GCP)

EXAMPLES:
  # Docker (default)
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git
  spinner spin --setup --image my-env --repo git@github.com:octocat/Hello-World.git --prompt "Fix the bug"

  # GCP
  spinner spin --backend gcp --image my-env --repo git@github.com:octocat/Hello-World.git \
    --project my-proj --zone us-central1-a --state-bucket my-bucket --prompt "Fix the bug"

  # With .spinner.json config, GCP flags can be omitted:
  spinner spin --image my-env --repo git@github.com:octocat/Hello-World.git --prompt "Fix the bug"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind general flags to Viper
			_ = viper.BindPFlag(flagImage, cmd.Flags().Lookup(flagImage))
			_ = viper.BindPFlag(flagRepo, cmd.Flags().Lookup(flagRepo))
			_ = viper.BindPFlag(flagPrompt, cmd.Flags().Lookup(flagPrompt))
			_ = viper.BindPFlag(flagBranch, cmd.Flags().Lookup(flagBranch))
			_ = viper.BindPFlag(flagMaxIterations, cmd.Flags().Lookup(flagMaxIterations))
			_ = viper.BindPFlag(flagRecreate, cmd.Flags().Lookup(flagRecreate))
			_ = viper.BindPFlag(flagSetup, cmd.Flags().Lookup(flagSetup))
			_ = viper.BindPFlag(flagBaseImage, cmd.Flags().Lookup(flagBaseImage))
			_ = viper.BindPFlag(flagDockerfile, cmd.Flags().Lookup(flagDockerfile))
			_ = viper.BindPFlag(flagWatch, cmd.Flags().Lookup(flagWatch))

			// Bind GCP flags to Viper
			_ = viper.BindPFlag(flagProject, cmd.Flags().Lookup(flagProject))
			_ = viper.BindPFlag(flagZone, cmd.Flags().Lookup(flagZone))
			_ = viper.BindPFlag(flagMachineType, cmd.Flags().Lookup(flagMachineType))
			_ = viper.BindPFlag(flagDiskSize, cmd.Flags().Lookup(flagDiskSize))
			_ = viper.BindPFlag(flagStateBucket, cmd.Flags().Lookup(flagStateBucket))

			// Resolve backend (CLI > env > config > default "docker")
			backend = resolveBackend(cmd)

			// Validate cross-backend flags
			if err := validateBackendFlags(cmd, backend); err != nil {
				return err
			}

			// Read values from Viper
			spinImage = viper.GetString(flagImage)
			spinRepo = viper.GetString(flagRepo)
			spinPrompt = viper.GetString(flagPrompt)
			spinBranch = viper.GetString(flagBranch)
			spinMaxIterations = viper.GetString(flagMaxIterations)
			spinRecreate = viper.GetBool(flagRecreate)
			spinSetup = viper.GetBool(flagSetup)
			spinBaseImage = viper.GetString(flagBaseImage)
			spinDockerfile = viper.GetString(flagDockerfile)
			spinWatch = viper.GetBool(flagWatch)

			if spinImage == "" {
				return fmt.Errorf("--image flag is required")
			}

			if spinRepo == "" {
				return fmt.Errorf("--repo flag is required")
			}

			// Docker-specific setup flag validation
			if backend == provider.BackendDocker {
				if err := validateDockerFlags(cmd); err != nil {
					return err
				}
			}

			// GCP-specific validation
			if backend == provider.BackendGCP {
				if err := validateRequiredGCPFlags(cmd); err != nil {
					return err
				}
			}

			// Create provider from factory
			p, err := f.Create(backend)
			if err != nil {
				return err
			}

			ctx := context.Background()

			// If --setup is provided, provision the environment first
			if spinSetup {
				setupName := strings.TrimPrefix(spinImage, "spinner:")

				setupOptions := map[string]string{
					flagBaseImage:  spinBaseImage,
					flagDockerfile: spinDockerfile,
				}

				if backend == provider.BackendGCP {
					setupOptions[flagProject] = viper.GetString(flagProject)
					setupOptions[flagZone] = viper.GetString(flagZone)
					setupOptions[flagStateBucket] = viper.GetString(flagStateBucket)

					mt := viper.GetString(flagMachineType)
					if mt == "" {
						mt = defaultMachineType
					}

					setupOptions[flagMachineType] = mt

					ds := viper.GetInt(flagDiskSize)
					if ds == 0 {
						ds = defaultDiskSize
					}

					setupOptions[flagDiskSize] = strconv.Itoa(ds)
				}

				if err := performSetup(ctx, p, provider.SetupConfig{
					Name:    setupName,
					Options: setupOptions,
				}); err != nil {
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

			if err := prerequisites.CheckEnvironmentVariables(); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
				return err
			}

			fmt.Println("✓ Prerequisites validated")

			createOptions := map[string]string{flagImage: spinImage}

			if backend == provider.BackendGCP {
				createOptions[flagProject] = viper.GetString(flagProject)
				createOptions[flagZone] = viper.GetString(flagZone)
				createOptions[flagStateBucket] = viper.GetString(flagStateBucket)

				mt := viper.GetString(flagMachineType)
				if mt == "" {
					mt = defaultMachineType
				}

				createOptions[flagMachineType] = mt

				ds := viper.GetInt(flagDiskSize)
				if ds == 0 {
					ds = defaultDiskSize
				}

				createOptions[flagDiskSize] = strconv.Itoa(ds)
			}

			createConfig := provider.CreateConfig{
				Repo:          spinRepo,
				Prompt:        spinPrompt,
				Branch:        spinBranch,
				MaxIterations: spinMaxIterations,
				Options:       createOptions,
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
				instance, err = p.Start(ctx, name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
					return err
				}

				fmt.Printf("✓ Instance restarted: %s\n", instance.Name)
				fmt.Println()
				fmt.Println("Note: Reusing existing instance. Use --recreate flag to force recreation.")
			}

			// Display backend-specific management commands
			fmt.Println()

			if backend == provider.BackendGCP {
				gcpProject := viper.GetString(flagProject)
				gcpZone := viper.GetString(flagZone)

				fmt.Printf("To access: gcloud compute ssh %s --project %s --zone %s\n", instance.Name, gcpProject, gcpZone)
				fmt.Printf("To stop:   gcloud compute instances stop %s --project %s --zone %s\n", instance.Name, gcpProject, gcpZone)
				fmt.Printf("To remove: gcloud compute instances delete %s --project %s --zone %s\n", instance.Name, gcpProject, gcpZone)
			} else {
				fmt.Printf("To access: docker exec -it %s bash\n", instance.Name)
				fmt.Printf("To stop: docker stop %s\n", instance.Name)
				fmt.Printf("To remove: docker rm %s\n", instance.Name)
			}

			// Enter watch mode if --watch flag is set
			if spinWatch {
				fmt.Println()
				fmt.Println("Entering watch mode...")

				return PerformWatch(ctx, p, instance.Name)
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
	cmd.Flags().BoolVar(&spinRecreate, flagRecreate, false, "Force recreation of existing instance (optional)")
	cmd.Flags().BoolVar(&spinSetup, flagSetup, false, "Build/rebuild the environment before spinning (optional)")
	cmd.Flags().StringVar(&backend, flagBackend, "", "Backend provider: docker, gcp (default: docker)")
	cmd.Flags().BoolVar(&spinWatch, flagWatch, false, "Enter watch mode after instance is ready (optional)")

	// Docker backend flags
	cmd.Flags().StringVar(&spinBaseImage, flagBaseImage, "", "Base Docker image (Docker backend, requires --setup)")
	cmd.Flags().StringVar(&spinDockerfile, flagDockerfile, "", "Path to custom Dockerfile (Docker backend, requires --setup)")

	// GCP backend flags
	cmd.Flags().StringVar(&project, flagProject, "", "GCP project ID (GCP backend)")
	cmd.Flags().StringVar(&zone, flagZone, "", "GCP zone (GCP backend)")
	cmd.Flags().StringVar(&machineType, flagMachineType, "", "VM machine type (GCP backend, default: e2-standard-2)")
	cmd.Flags().IntVar(&diskSize, flagDiskSize, 0, "Boot disk size in GB (GCP backend, default: 30)")
	cmd.Flags().StringVar(&stateBucket, flagStateBucket, "", "GCS bucket for state persistence (GCP backend)")

	return cmd
}
