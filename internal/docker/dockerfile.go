package docker

import (
	"os"
	"path/filepath"
	"strings"
)

type DockerfileConfig struct {
	BaseImage string
}

func GenerateDockerfile(config DockerfileConfig) (string, error) {
	templatePath := filepath.Join("templates", "docker", "extending.template")
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	template := string(templateBytes)
	return strings.ReplaceAll(template, "${BASE_IMAGE}", config.BaseImage), nil
}
