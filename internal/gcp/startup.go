package gcp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rickihastings/spinner/internal/util"
)

// LoadBakeScript reads the GCP bake startup script from templates/scripts/gcp_bake.sh.
// This script is passed as metadata.startup-script when creating the temporary bake VM.
func LoadBakeScript() (string, error) {
	scriptPath, err := util.ResolveTemplatePath(filepath.Join("templates", "scripts", "gcp_bake.sh"))
	if err != nil {
		return "", fmt.Errorf("failed to find bake script: %w", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read bake script: %w", err)
	}

	return string(data), nil
}
