package docker

import (
	"os"
	"path/filepath"
	"strings"
)

// DockerfileConfig contains configuration for generating a Dockerfile.
type DockerfileConfig struct {
	BaseImage string
}

// GenerateDockerfile generates a Dockerfile from a template with the given configuration.
func GenerateDockerfile(config DockerfileConfig) (string, error) {
	templatePath, err := resolveTemplatePath(filepath.Join("templates", "docker", "extending.template"))
	if err != nil {
		return "", err
	}

	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	template := string(templateBytes)

	return strings.ReplaceAll(template, "${BASE_IMAGE}", config.BaseImage), nil
}
