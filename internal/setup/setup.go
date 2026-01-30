package setup

import (
	"fmt"
	"os"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/prerequisites"
)

// Config contains configuration for setup operations
type Config struct {
	Name       string
	BaseImage  string
	Dockerfile string
}

// PerformSetup executes the complete setup workflow:
// 1. Check prerequisites (Docker, git, claude-code)
// 2. Validate Dockerfile path if provided
// 3. Build Docker image
func PerformSetup(config Config) error {
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

	buildConfig := docker.BuildConfig{
		Name:       config.Name,
		BaseImage:  config.BaseImage,
		Dockerfile: config.Dockerfile,
	}

	if err := docker.BuildImage(buildConfig); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())
		return err
	}

	fmt.Printf("✓ Docker image built successfully: spinner:%s\n", config.Name)

	return nil
}
