package gcp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/rickihastings/spinner/internal/util"
)

// bakeTemplateData holds the template variables for the GCP bake script.
type bakeTemplateData struct {
	// BakeScript is the contents of the user's custom bake script (if any).
	// Injected inline after core tooling is installed, before shutdown.
	BakeScript string
}

// loadBakeScript reads the GCP bake startup script template from
// templates/scripts/gcp_bake.sh and renders it with the given custom
// bake script contents. If customBakeScript is empty, the template block
// is omitted and the default bake runs unchanged.
func loadBakeScript(customBakeScript string) (string, error) {
	scriptPath, err := util.ResolveTemplatePath(filepath.Join("templates", "scripts", "gcp_bake.sh"))
	if err != nil {
		return "", fmt.Errorf("failed to find bake script: %w", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read bake script: %w", err)
	}

	tmpl, err := template.New("gcp_bake").Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("failed to parse bake script template: %w", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, bakeTemplateData{
		BakeScript: customBakeScript,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render bake script template: %w", err)
	}

	return buf.String(), nil
}

// loadBakeScriptFile reads a custom bake script file and returns its contents.
// Returns empty string if path is empty. Returns an error if the file cannot be read.
func loadBakeScriptFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read custom bake script %s: %w", path, err)
	}

	return string(data), nil
}

// loadStartupScript reads the standard startup.sh template used inside containers/VMs.
// This script handles repo cloning, branch checkout, and spinner exec invocation.
func loadStartupScript() (string, error) {
	scriptPath, err := util.ResolveTemplatePath(filepath.Join("templates", "scripts", "startup.sh"))
	if err != nil {
		return "", fmt.Errorf("failed to find startup script: %w", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read startup script: %w", err)
	}

	return string(data), nil
}

// loadRuntimeScript reads the GCP runtime startup script template from
// templates/scripts/gcp_runtime.sh. This script reads instance metadata,
// sets environment variables, and delegates to startup.sh.
func loadRuntimeScript() (string, error) {
	scriptPath, err := util.ResolveTemplatePath(filepath.Join("templates", "scripts", "gcp_runtime.sh"))
	if err != nil {
		return "", fmt.Errorf("failed to find runtime script: %w", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read runtime script: %w", err)
	}

	return string(data), nil
}

// loadInstallSpinnerScript reads the shared install_spinner.sh script.
// This script handles binary installation from either GitHub releases or local dev.
func loadInstallSpinnerScript() (string, error) {
	scriptPath, err := util.ResolveTemplatePath(filepath.Join("templates", "scripts", "install_spinner.sh"))
	if err != nil {
		return "", fmt.Errorf("failed to find install script: %w", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read install script: %w", err)
	}

	return string(data), nil
}
