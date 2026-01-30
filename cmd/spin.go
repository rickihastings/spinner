package cmd

import (
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/setup"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

var spinCmd = &cobra.Command{
	Use:   "spin",
	Short: "Spin up a development container from a pre-built image",
	Long: `Spin up a development container from a pre-built image

SPIN OPTIONS:
  --image <image>            Docker image to use (required)
  --repo <repo>              Git repository URL (required)
  --prompt <prompt>          Task prompt for ralph-loop (optional)
  --branch <branch>          Git branch to checkout (optional)
  --max-iterations <num>     Maximum iterations for ralph-loop (optional, default: 100)
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
		viper.BindPFlag("image", cmd.Flags().Lookup("image"))
		viper.BindPFlag("repo", cmd.Flags().Lookup("repo"))
		viper.BindPFlag("prompt", cmd.Flags().Lookup("prompt"))
		viper.BindPFlag("branch", cmd.Flags().Lookup("branch"))
		viper.BindPFlag("max-iterations", cmd.Flags().Lookup("max-iterations"))
		viper.BindPFlag("recreate", cmd.Flags().Lookup("recreate"))
		viper.BindPFlag("setup", cmd.Flags().Lookup("setup"))
		viper.BindPFlag("base-image", cmd.Flags().Lookup("base-image"))
		viper.BindPFlag("dockerfile", cmd.Flags().Lookup("dockerfile"))

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

		// If --setup is provided, build the image first
		if spinSetup {
			// Remove "spinner:" prefix from image if present for setup name
			setupName := spinImage
			if len(setupName) > 8 && setupName[:8] == "spinner:" {
				setupName = setupName[8:]
			}

			// Perform setup using shared logic
			if err := setup.PerformSetup(setup.Config{
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

		validationResult := docker.ValidatePrerequisites(config)
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
		containerStatus := docker.CheckContainerExists(containerName)

		// Handle --recreate flag: remove existing container and create fresh
		if spinRecreate && containerStatus != docker.StatusNone {
			removeResult := docker.RemoveContainer(containerName)
			if !removeResult.Success {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", removeResult.Error)
				return fmt.Errorf("failed to remove container: %s", removeResult.Error)
			}
			// After removal, container doesn't exist
			containerStatus = docker.StatusNone
		}

		var action docker.ReuseAction

		if containerStatus == docker.StatusNone {
			// Create new container
			fmt.Printf("Creating container: %s\n", containerName)
			fmt.Println("Cloning repository...")

			dockerArgs, err := docker.BuildDockerRunCommand(config, containerName, validationResult.HasNpmrc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
				return err
			}
			runResult := docker.ExecuteDockerRun(dockerArgs, containerName)
			if !runResult.Success {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", runResult.Error)
				return fmt.Errorf("failed to run container: %s", runResult.Error)
			}

			// Verify container is running
			statusResult := docker.VerifyContainerStatus(containerName)
			if !statusResult.Success {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", statusResult.Error)
				return fmt.Errorf("container verification failed: %s", statusResult.Error)
			}

			action = docker.ActionCreated
		} else if containerStatus == docker.StatusRunning {
			// Reuse running container
			action = docker.ActionReused
		} else if containerStatus == docker.StatusStopped {
			// Restart stopped container
			restartResult := docker.RestartContainer(containerName)
			if !restartResult.Success {
				fmt.Fprintf(os.Stderr, "✗ Error: %s\n", restartResult.Error)
				return fmt.Errorf("failed to restart container: %s", restartResult.Error)
			}

			// Verify container is running after restart
			statusResult := docker.VerifyContainerStatus(containerName)
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

func init() {
	spinCmd.Flags().StringVar(&spinImage, "image", "", "Docker image to use (required)")
	spinCmd.Flags().StringVar(&spinRepo, "repo", "", "Git repository URL (required)")
	spinCmd.Flags().StringVar(&spinPrompt, "prompt", "", "Task prompt for ralph-loop (optional)")
	spinCmd.Flags().StringVar(&spinBranch, "branch", "", "Git branch to checkout (optional)")
	spinCmd.Flags().StringVar(&spinMaxIterations, "max-iterations", "", "Maximum iterations for ralph-loop (optional, default: 100)")
	spinCmd.Flags().BoolVar(&spinRecreate, "recreate", false, "Force recreation of existing container (optional)")
	spinCmd.Flags().BoolVar(&spinSetup, "setup", false, "Build/rebuild the Docker image before spinning (optional)")
	spinCmd.Flags().StringVar(&spinBaseImage, "base-image", "", "Base Docker image (optional, default: ubuntu:22.04, requires --setup)")
	spinCmd.Flags().StringVar(&spinDockerfile, "dockerfile", "", "Path to custom Dockerfile (optional, requires --setup)")

	rootCmd.AddCommand(spinCmd)
}
