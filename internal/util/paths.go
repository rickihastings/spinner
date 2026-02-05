package util

import (
	"os"
	"path/filepath"
)

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
