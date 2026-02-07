package gcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBakeScript(t *testing.T) {
	script, err := LoadBakeScript("")
	assert.NoError(t, err)
	assert.NotEmpty(t, script)
}

func TestLoadBakeScriptContent(t *testing.T) {
	script, err := LoadBakeScript("")
	assert.NoError(t, err)

	// Script should start with a shebang
	assert.True(t, strings.HasPrefix(script, "#!/bin/bash"), "script should start with shebang")

	// Script should contain key installation steps
	assert.Contains(t, script, "apt-get", "script should install packages")
	assert.Contains(t, script, "spinner", "script should reference spinner")
	assert.Contains(t, script, "SPINNER_BAKE_COMPLETE", "script should signal completion")
	assert.Contains(t, script, "shutdown -h now", "script should shut down after baking")
}

func TestLoadBakeScriptIsExecutable(t *testing.T) {
	script, err := LoadBakeScript("")
	assert.NoError(t, err)

	// Should have set -e for fail-fast behavior
	assert.Contains(t, script, "set -e", "script should use set -e for safety")
}

func TestLoadBakeScriptWithoutCustomScript(t *testing.T) {
	script, err := LoadBakeScript("")
	assert.NoError(t, err)

	// Without a custom script, should NOT contain the custom bake script markers
	assert.NotContains(t, script, "Running custom bake script...")
	assert.NotContains(t, script, "Custom bake script completed.")
}

func TestLoadBakeScriptWithCustomScript(t *testing.T) {
	customScript := `apt-get install -y nodejs
npm install -g typescript`

	script, err := LoadBakeScript(customScript)
	assert.NoError(t, err)

	// Should contain the custom script content inline
	assert.Contains(t, script, "Running custom bake script...")
	assert.Contains(t, script, "apt-get install -y nodejs")
	assert.Contains(t, script, "npm install -g typescript")
	assert.Contains(t, script, "Custom bake script completed.")

	// Custom script should appear BEFORE the completion signal
	bakeCompleteIdx := strings.Index(script, "SPINNER_BAKE_COMPLETE")
	customScriptIdx := strings.Index(script, "apt-get install -y nodejs")
	assert.Greater(t, bakeCompleteIdx, customScriptIdx,
		"custom script should run before bake completion signal")

	// Custom script should appear AFTER core tooling
	aptGetIdx := strings.Index(script, "apt-get update")
	assert.Greater(t, customScriptIdx, aptGetIdx,
		"custom script should run after core tooling installation")
}

func TestLoadBakeScriptPreservesShellStructure(t *testing.T) {
	customScript := `echo "installing custom tools"
curl -fsSL https://example.com/install.sh | bash`

	script, err := LoadBakeScript(customScript)
	assert.NoError(t, err)

	// Should still start with shebang
	assert.True(t, strings.HasPrefix(script, "#!/bin/bash"))

	// Should still end with shutdown
	assert.Contains(t, script, "shutdown -h now")

	// Custom script content preserved
	assert.Contains(t, script, `echo "installing custom tools"`)
}

func TestLoadBakeScriptFileEmptyPath(t *testing.T) {
	content, err := LoadBakeScriptFile("")
	assert.NoError(t, err)
	assert.Empty(t, content)
}

func TestLoadBakeScriptFileReadSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "custom-bake.sh")

	scriptContent := "#!/bin/bash\napt-get install -y python3\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	require.NoError(t, err)

	content, err := LoadBakeScriptFile(scriptPath)
	assert.NoError(t, err)
	assert.Equal(t, scriptContent, content)
}

func TestLoadBakeScriptFileNotFound(t *testing.T) {
	_, err := LoadBakeScriptFile("/nonexistent/path/custom-bake.sh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read custom bake script")
}
