package docker

import (
	"os"

	"github.com/rickihastings/spinner/internal/util"
)

// BuildConfig contains configuration for building a Docker image.
type BuildConfig struct {
	Name      string
	ExtraArgs []string
}

// buildFile defines a file to copy into the Docker build context
type buildFile struct {
	// File contents
	file string
	// Destination path relative to the build context templates directory
	dst string
}

// buildFiles lists all files to copy into the Docker build context
var buildFiles = []buildFile{
	{file: util.LoadStartupScript(), dst: "scripts/startup.sh"},
	{file: util.LoadInstallSpinnerScript(), dst: "scripts/install_spinner.sh"},
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}
