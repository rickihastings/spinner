package exec

import (
	"fmt"
	"os"
	"strconv"
)

// Config represents the configuration for the exec command.
// All values are loaded from environment variables.
type Config struct {
	Prompt        string
	MaxIterations int
	Branch        string
	LogDir        string
	ContainerName string
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*Config, error) {
	config := &Config{
		Prompt:        os.Getenv("PROMPT"),
		Branch:        os.Getenv("BRANCH"),
		LogDir:        os.Getenv("LOG_DIR"),
		ContainerName: os.Getenv("CONTAINER_NAME"),
	}

	// Parse MAX_ITERATIONS
	maxIterStr := os.Getenv("MAX_ITERATIONS")
	if maxIterStr == "" {
		return nil, fmt.Errorf("MAX_ITERATIONS is required")
	}

	maxIter, err := strconv.Atoi(maxIterStr)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_ITERATIONS: %w", err)
	}

	if maxIter <= 0 {
		return nil, fmt.Errorf("MAX_ITERATIONS must be greater than 0")
	}

	config.MaxIterations = maxIter

	// Validate required fields
	if config.Prompt == "" {
		return nil, fmt.Errorf("PROMPT is required")
	}

	return config, nil
}
