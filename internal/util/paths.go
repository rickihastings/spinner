package util

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed templates/scripts/startup.sh
var startupScript string

//go:embed templates/scripts/install_spinner.sh
var installSpinnerScript string

// FindProjectRoot walks up the directory tree to find the project root
// (indicated by the presence of go.mod file)
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check if go.mod exists in current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding project root
			return "", os.ErrNotExist
		}

		dir = parent
	}
}

// ResolveTemplatePath resolves a template path relative to the project root
func ResolveTemplatePath(relativePath string) (string, error) {
	// First try relative to current directory
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath, nil
	}

	// Try relative to project root
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return "", err
	}

	absolutePath := filepath.Join(projectRoot, relativePath)
	if _, err := os.Stat(absolutePath); err != nil {
		return "", err
	}

	return absolutePath, nil
}

// LoadStartupScript reads the standard startup.sh template used inside containers/VMs.
// This script handles repo cloning, branch checkout, and spinner exec invocation.
func LoadStartupScript() string {
	return startupScript
}

// LoadInstallSpinnerScript reads the shared install_spinner.sh script.
// This script handles binary installation from either GitHub releases or local dev.
func LoadInstallSpinnerScript() string {
	return installSpinnerScript
}
