package gcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadBakeScript(t *testing.T) {
	script, err := LoadBakeScript()
	assert.NoError(t, err)
	assert.NotEmpty(t, script)
}

func TestLoadBakeScriptContent(t *testing.T) {
	script, err := LoadBakeScript()
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
	script, err := LoadBakeScript()
	assert.NoError(t, err)

	// Should have set -e for fail-fast behavior
	assert.Contains(t, script, "set -e", "script should use set -e for safety")
}
