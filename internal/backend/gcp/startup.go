package gcp

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"text/template"
)

//go:embed templates/scripts/gcp_bake.sh
var bakeScript string

//go:embed templates/scripts/gcp_runtime.sh
var runtimeScript string

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
	tmpl, err := template.New("gcp_bake").Parse(bakeScript)
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

// loadRuntimeScript reads the GCP runtime startup script template from
// templates/scripts/gcp_runtime.sh. This script reads instance metadata,
// sets environment variables, and delegates to startup.sh.
func loadRuntimeScript() string {
	return runtimeScript
}
