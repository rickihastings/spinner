package exec

import (
	"fmt"

	"github.com/spf13/viper"
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

// LoadConfig loads configuration from environment variables using viper.
func LoadConfig() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	config := &Config{
		Prompt:        v.GetString("PROMPT"),
		MaxIterations: v.GetInt("MAX_ITERATIONS"),
		Branch:        v.GetString("BRANCH"),
		LogDir:        v.GetString("LOG_DIR"),
		ContainerName: v.GetString("CONTAINER_NAME"),
	}

	// Validate MAX_ITERATIONS
	if config.MaxIterations == 0 {
		return nil, fmt.Errorf("MAX_ITERATIONS is required")
	}

	if config.MaxIterations < 0 {
		return nil, fmt.Errorf("MAX_ITERATIONS must be greater than 0")
	}

	// Validate required fields
	if config.Prompt == "" {
		return nil, fmt.Errorf("PROMPT is required")
	}

	return config, nil
}
