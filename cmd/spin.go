package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/prerequisites"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// spinCmd is the production spin command using Provider
var spinCmd = NewSpinCommand(docker.NewDockerProvider(docker.NewRealDockerClient()))

func init() {
	rootCmd.AddCommand(spinCmd)
}

// NewSpinCommand creates a new spin command with the given Provider.
// This constructor enables dependency injection for testing.
func NewSpinCommand(p provider.Provider) *cobra.Command {
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
	)

	cmd := &cobra.Command{
		Use:   "spin",
		Short: "Spin up a development container from a pre-built image",
		Long: `Spin up a development container from a pre-built image

SPIN OPTIONS:
  --image <image>            Docker image to use (required)
  --repo <repo>              Git repository URL (required)
  --prompt <prompt>          Task prompt for autonomous execution (optional)
  --branch <branch>          Git branch to checkout (optional)
  --max-iterations <num>     Maximum iterations for autonomous execution (optional, default: 100)
  --recreate                 Force recreation of existing container (optional)
  --watch                    Enter watch mode after container is ready (optional)

SETUP OPTIONS (use with --setup flag):
  --setup                    Build/rebuild the Docker image before spinning (optional)
  --base-image <image>       Base Docker image (optional, default: ubuntu:22.04, requires --setup)
  --dockerfile <path>        Path to custom Dockerfile (optional, requires --setup, mutually exclusive with --base-image)

EXAMPLES:
  # Basic spin with existing image
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git

  # Spin with setup (builds image first)
  spinner spin --setup --image my-env --repo git@github.com:octocat/Hello-World.git

  # Setup with custom base image
  spinner spin --setup --image my-env --base-image node:20-bullseye --repo git@github.com:octocat/Hello-World.git

  # Setup with custom Dockerfile
  spinner spin --setup --image my-env --dockerfile ./Dockerfile.custom --repo git@github.com:octocat/Hello-World.git

  # Other spin options work with setup
  spinner spin --setup --image my-env --repo git@github.com:octocat/Hello-World.git --prompt "Implement feature X"
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git --recreate

  # Watch mode after spinning
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git --watch

Note: When --setup is used, the image is always rebuilt (no caching). The --image value becomes the setup name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = viper.BindPFlag("image", cmd.Flags().Lookup("image"))
			_ = viper.BindPFlag("repo", cmd.Flags().Lookup("repo"))
			_ = viper.BindPFlag("prompt", cmd.Flags().Lookup("prompt"))
			_ = viper.BindPFlag("branch", cmd.Flags().Lookup("branch"))
			_ = viper.BindPFlag("max-iterations", cmd.Flags().Lookup("max-iterations"))
			_ = viper.BindPFlag("recreate", cmd.Flags().Lookup("recreate"))
			_ = viper.BindPFlag("setup", cmd.Flags().Lookup("setup"))
			_ = viper.BindPFlag("base-image", cmd.Flags().Lookup("base-image"))
			_ = viper.BindPFlag("dockerfile", cmd.Flags().Lookup("dockerfile"))
			_ = viper.BindPFlag("watch", cmd.Flags().Lookup("watch"))

			spinImage = viper.GetString("image")
			spinRepo = viper.GetString("repo")
			spinPrompt = viper.GetString("prompt")
			spinBranch = viper.GetString("branch")
			spinMaxIterations = viper.GetString("max-iterations")
			spinRecreate = viper.GetBool("recreate")
			spinSetup = viper.GetBool("setup")
			spinBaseImage = viper.GetString("base-image")
			spinDockerfile = viper.GetString("dockerfile")
			spinWatch = viper.GetBool("watch")

			if spinImage == "" {
				return fmt.Errorf("--image flag is required")
			}

			if spinRepo == "" {
				return fmt.Errorf("--repo flag is required")
			}

			if !spinSetup && spinBaseImage != "" {
				fmt.Fprintln(os.Stderr, "Error: --base-image requires --setup flag")
				return fmt.Errorf("--base-image requires --setup flag")
			}

			if !spinSetup && spinDockerfile != "" {
				fmt.Fprintln(os.Stderr, "Error: --dockerfile requires --setup flag")
				return fmt.Errorf("--dockerfile requires --setup flag")
			}

			if spinSetup && spinBaseImage != "" && spinDockerfile != "" {
				fmt.Fprintln(os.Stderr, "Error: --base-image and --dockerfile are mutually exclusive")
				fmt.Fprintln(os.Stderr, "Please provide only one of these flags")

				return fmt.Errorf("mutually exclusive flags provided")
			}

			ctx := context.Background()

			// If --setup is provided, provision the environment first
			if spinSetup {
				setupName := strings.TrimPrefix(spinImage, "spinner:")

				if err := performSetup(ctx, p, provider.SetupConfig{
					Name:    setupName,
					Options: map[string]string{"base-image": spinBaseImage, "dockerfile": spinDockerfile},
				}); err != nil {
					return err
				}

				spinImage = "spinner:" + setupName
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

			createConfig := provider.CreateConfig{
				Repo:          spinRepo,
				Prompt:        spinPrompt,
				Branch:        spinBranch,
				MaxIterations: spinMaxIterations,
				Options:       map[string]string{"image": spinImage},
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

			// Display instance management commands
			fmt.Println()
			fmt.Printf("To access: docker exec -it %s bash\n", instance.Name)
			fmt.Printf("To stop: docker stop %s\n", instance.Name)
			fmt.Printf("To remove: docker rm %s\n", instance.Name)

			// Enter watch mode if --watch flag is set
			if spinWatch {
				fmt.Println()
				fmt.Println("Entering watch mode...")

				return PerformWatch(ctx, p, instance.Name)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&spinImage, "image", "", "Docker image to use (required)")
	cmd.Flags().StringVar(&spinRepo, "repo", "", "Git repository URL (required)")
	cmd.Flags().StringVar(&spinPrompt, "prompt", "", "Task prompt for autonomous execution (optional)")
	cmd.Flags().StringVar(&spinBranch, "branch", "", "Git branch to checkout (optional)")
	cmd.Flags().StringVar(&spinMaxIterations, "max-iterations", "", "Maximum iterations for autonomous execution (optional, default: 100)")
	cmd.Flags().BoolVar(&spinRecreate, "recreate", false, "Force recreation of existing container (optional)")
	cmd.Flags().BoolVar(&spinSetup, "setup", false, "Build/rebuild the Docker image before spinning (optional)")
	cmd.Flags().StringVar(&spinBaseImage, "base-image", "", "Base Docker image (optional, default: ubuntu:22.04, requires --setup)")
	cmd.Flags().StringVar(&spinDockerfile, "dockerfile", "", "Path to custom Dockerfile (optional, requires --setup)")
	cmd.Flags().BoolVar(&spinWatch, "watch", false, "Enter watch mode after container is ready (optional)")

	return cmd
}
