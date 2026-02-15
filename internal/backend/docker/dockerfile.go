package docker

import (
	_ "embed"
	"strings"
)

//go:embed templates/docker/extending.template
var extendingTemplate string

// dockerfileConfig contains configuration for generating a Dockerfile.
type dockerfileConfig struct {
	BaseImage string
}

// generateDockerfile generates a Dockerfile from a template with the given configuration.
func generateDockerfile(config dockerfileConfig) (string, error) {
	return strings.ReplaceAll(extendingTemplate, "${BASE_IMAGE}", config.BaseImage), nil
}
