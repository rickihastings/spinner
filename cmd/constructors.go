package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/exec"
	"github.com/rickihastings/spinner/internal/prerequisites"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewSetupCommand creates a new setup command with the given DockerClient.
// This constructor enables dependency injection for testing.
func NewSetupCommand(client docker.DockerClient) *cobra.Command {
	var (
		setupName       string
		setupBaseImage  string
		setupDockerfile string
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Build a Docker sandbox image",
		Long: `Build a Docker sandbox image with custom base image or Dockerfile

SETUP OPTIONS:
  --name <name>              Name for the Docker image (required)
  --base-image <image>       Base Docker image (optional, default: ubuntu:22.04)
  --dockerfile <path>        Path to custom Dockerfile (optional, mutually exclusive with --base-image)

EXAMPLES:
  spinner setup --name my-sandbox
  spinner setup --name my-sandbox --base-image ubuntu:22.04
  spinner setup --name node-env --base-image node:20-bullseye
  spinner setup --name custom-env --dockerfile ./Dockerfile.custom`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind flags to viper - this allows environment variables to override flag values
			_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))
			_ = viper.BindPFlag("base-image", cmd.Flags().Lookup("base-image"))
			_ = viper.BindPFlag("dockerfile", cmd.Flags().Lookup("dockerfile"))

			// Get values from viper (respects env vars and flags)
			setupName = viper.GetString("name")
			setupBaseImage = viper.GetString("base-image")
			setupDockerfile = viper.GetString("dockerfile")

			// Validate required flag
			if setupName == "" {
				fmt.Fprintln(os.Stderr, "Error: Missing required flag: --name")
				fmt.Fprintln(os.Stderr, "Usage: spinner setup --name <name> [--base-image <image> | --dockerfile <path>]")

				return fmt.Errorf("missing required flag: --name")
			}

			// Validate mutually exclusive flags
			if setupBaseImage != "" && setupDockerfile != "" {
				fmt.Fprintln(os.Stderr, "Error: --base-image and --dockerfile are mutually exclusive")
				fmt.Fprintln(os.Stderr, "Please provide only one of these flags")

				return fmt.Errorf("mutually exclusive flags provided")
			}

			// Perform setup using the provided client
			return performSetupWithClient(context.Background(), client, docker.BuildConfig{
				Name:       setupName,
				BaseImage:  setupBaseImage,
				Dockerfile: setupDockerfile,
			})
		},
	}

	cmd.Flags().StringVar(&setupName, "name", "", "Name for the Docker image (required)")
	cmd.Flags().StringVar(&setupBaseImage, "base-image", "", "Base Docker image (optional, default: ubuntu:22.04)")
	cmd.Flags().StringVar(&setupDockerfile, "dockerfile", "", "Path to custom Dockerfile (optional)")

	return cmd
}

// performSetupWithClient executes the setup workflow with a provided DockerClient.
func performSetupWithClient(ctx context.Context, client docker.DockerClient, config docker.BuildConfig) error {
	// Check prerequisites
	fmt.Println("Checking prerequisites...")

	if err := prerequisites.CheckPrerequisites(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
		return err
	}

	// Validate Dockerfile path if provided
	if config.Dockerfile != "" {
		if _, err := os.Stat(config.Dockerfile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "✗ Error: Dockerfile not found at path: %s\n", config.Dockerfile)
			return fmt.Errorf("dockerfile not found at path: %s", config.Dockerfile)
		}
	}

	// Build the image
	fmt.Printf("✓ Prerequisites checked\n")
	fmt.Printf("Building Docker image: spinner:%s\n", config.Name)

	if err := client.BuildImage(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
		return err
	}

	fmt.Printf("✓ Docker image built successfully: spinner:%s\n", config.Name)

	return nil
}

// NewSpinCommand creates a new spin command with the given DockerClient.
// This constructor enables dependency injection for testing.
func NewSpinCommand(client docker.DockerClient) *cobra.Command {
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

Note: When --setup is used, the image is always rebuilt (no caching). The --image value becomes the setup name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind flags to viper - this allows environment variables to override flag values
			_ = viper.BindPFlag("image", cmd.Flags().Lookup("image"))
			_ = viper.BindPFlag("repo", cmd.Flags().Lookup("repo"))
			_ = viper.BindPFlag("prompt", cmd.Flags().Lookup("prompt"))
			_ = viper.BindPFlag("branch", cmd.Flags().Lookup("branch"))
			_ = viper.BindPFlag("max-iterations", cmd.Flags().Lookup("max-iterations"))
			_ = viper.BindPFlag("recreate", cmd.Flags().Lookup("recreate"))
			_ = viper.BindPFlag("setup", cmd.Flags().Lookup("setup"))
			_ = viper.BindPFlag("base-image", cmd.Flags().Lookup("base-image"))
			_ = viper.BindPFlag("dockerfile", cmd.Flags().Lookup("dockerfile"))

			// Get values from viper (respects env vars and flags)
			spinImage = viper.GetString("image")
			spinRepo = viper.GetString("repo")
			spinPrompt = viper.GetString("prompt")
			spinBranch = viper.GetString("branch")
			spinMaxIterations = viper.GetString("max-iterations")
			spinRecreate = viper.GetBool("recreate")
			spinSetup = viper.GetBool("setup")
			spinBaseImage = viper.GetString("base-image")
			spinDockerfile = viper.GetString("dockerfile")

			// Validate required flags
			if spinImage == "" {
				return fmt.Errorf("--image flag is required")
			}

			if spinRepo == "" {
				return fmt.Errorf("--repo flag is required")
			}

			// Validate setup-related flags
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

			// If --setup is provided, build the image first
			if spinSetup {
				// Remove "spinner:" prefix from image if present for setup name
				setupName := spinImage
				if len(setupName) > 8 && setupName[:8] == "spinner:" {
					setupName = setupName[8:]
				}

				// Perform setup using the provided client
				if err := performSetupWithClient(ctx, client, docker.BuildConfig{
					Name:       setupName,
					BaseImage:  spinBaseImage,
					Dockerfile: spinDockerfile,
				}); err != nil {
					return err
				}

				// Update spinImage to use the built image tag
				spinImage = "spinner:" + setupName
			}

			// Validate prerequisites
			fmt.Println("Validating prerequisites...")

			config := docker.SpinConfig{
				Image:         spinImage,
				Repo:          spinRepo,
				Prompt:        spinPrompt,
				Branch:        spinBranch,
				MaxIterations: spinMaxIterations,
				Recreate:      spinRecreate,
			}

			validationResult := docker.ValidatePrerequisitesWithClient(ctx, client, config)
			if !validationResult.Valid {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", validationResult.Error)
				return fmt.Errorf("validation failed: %s", validationResult.Error)
			}

			// Generate container name
			containerName := docker.GenerateContainerName(config)

			fmt.Println("✓ Prerequisites validated")

			for _, warning := range validationResult.Warnings {
				fmt.Printf("⚠ Warning: %s\n", warning)
			}

			// Check if container already exists
			containerStatus, _ := client.ContainerExists(ctx, containerName)

			// Handle --recreate flag: remove existing container and create fresh
			if spinRecreate && containerStatus != docker.StatusNone {
				removeResult, _ := client.RemoveContainer(ctx, containerName)
				if !removeResult.Success {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", removeResult.Error)
					return fmt.Errorf("failed to remove container: %s", removeResult.Error)
				}
				// After removal, container doesn't exist
				containerStatus = docker.StatusNone
			}

			var action docker.ReuseAction

			switch containerStatus {
			case docker.StatusNone:
				// Create new container
				fmt.Printf("Creating container: %s\n", containerName)
				fmt.Println("Cloning repository...")

				dockerArgs, err := docker.BuildDockerRunCommand(config, containerName, validationResult.HasNpmrc)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
					return err
				}

				runResult, _ := client.RunContainer(ctx, dockerArgs, containerName)
				if !runResult.Success {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", runResult.Error)
					return fmt.Errorf("failed to run container: %s", runResult.Error)
				}

				// Verify container is running
				statusResult, _ := client.VerifyContainerStatus(ctx, containerName)
				if !statusResult.Success {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", statusResult.Error)
					return fmt.Errorf("container verification failed: %s", statusResult.Error)
				}

				action = docker.ActionCreated
			case docker.StatusRunning:
				// Reuse running container
				action = docker.ActionReused
			case docker.StatusStopped:
				// Restart stopped container
				restartResult, _ := client.RestartContainer(ctx, containerName)
				if !restartResult.Success {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", restartResult.Error)
					return fmt.Errorf("failed to restart container: %s", restartResult.Error)
				}

				// Verify container is running after restart
				statusResult, _ := client.VerifyContainerStatus(ctx, containerName)
				if !statusResult.Success {
					fmt.Fprintf(os.Stderr, "✗ Error: %s\n", statusResult.Error)
					return fmt.Errorf("container verification failed: %s", statusResult.Error)
				}

				action = docker.ActionRestarted
			}

			// Display success message based on action
			switch action {
			case docker.ActionCreated:
				fmt.Printf("✓ Container created successfully: %s\n", containerName)
			case docker.ActionRestarted:
				fmt.Printf("✓ Container restarted: %s\n", containerName)
			case docker.ActionReused:
				fmt.Printf("✓ Reusing running container: %s\n", containerName)
			}

			// Display management note for reuse cases
			if action != docker.ActionCreated {
				fmt.Println()
				fmt.Println("Note: Reusing existing container. Use --recreate flag to force recreation.")
			}

			// Display container management commands
			fmt.Println()
			fmt.Printf("To access: docker exec -it %s bash\n", containerName)
			fmt.Printf("To stop: docker stop %s\n", containerName)
			fmt.Printf("To remove: docker rm %s\n", containerName)

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

	return cmd
}

// NewExecCommand creates a new exec command.
// This command runs inside Docker containers and executes the iteration loop.
func NewExecCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute the autonomous iteration loop (runs inside containers)",
		Long: `Execute the autonomous iteration loop inside a Docker container.

This command is designed to run inside Docker containers created by 'spinner spin'.
It reads configuration from environment variables and manages an iteration loop
that interacts with Claude CLI to complete tasks.

ENVIRONMENT VARIABLES:
  PROMPT             Task prompt for the iteration loop (required)
  MAX_ITERATIONS     Maximum number of iterations (required)
  BRANCH             Git branch name (optional)
  LOG_DIR            Directory for log files (optional)
  STATE_DIR          Directory for state file (optional, defaults to /state)

STATE MANAGEMENT:
  State is persisted to ${STATE_DIR}/state.json (mounted from host)
  This allows iteration progress to survive container restarts.

EXAMPLES:
  # Typically called automatically by container startup script
  spinner exec

  # Can be called manually with environment variables
  PROMPT="Fix bug" MAX_ITERATIONS=10 spinner exec`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration from environment variables
			config, err := exec.LoadConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
				return err
			}

			// Build state file path from STATE_DIR (defaults to /state)
			statePath := filepath.Join(config.StateDir, "state.json")

			// Load or initialize state
			state, err := exec.LoadState(statePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
				return err
			}

			// Create context with signal handling for Ctrl+C
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Set up signal handling
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				fmt.Println("\nReceived interrupt signal...")
				cancel()
			}()

			// Create runner and execute loop
			runner := exec.NewRunner(config, state, statePath)
			exitCode := runner.Run(ctx)

			// Exit with the appropriate code
			os.Exit(exitCode)

			return nil
		},
	}

	return cmd
}
