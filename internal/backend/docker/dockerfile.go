package docker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rickihastings/spinner/internal/util"
)

// dockerfileConfig contains configuration for generating a Dockerfile.
type dockerfileConfig struct {
	BaseImage string
}

// generateDockerfile generates a Dockerfile from a template with the given configuration.
func generateDockerfile(config dockerfileConfig) (string, error) {
	templatePath, err := util.ResolveTemplatePath(filepath.Join("templates", "docker", "extending.template"))
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
