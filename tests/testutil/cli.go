package testutil

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// BinaryPath is the path to the compiled test binary
var BinaryPath = "../../dist/spinner"

// getSharedBinaryPath is implemented in integration package to access the global binary path.
var getSharedBinaryPath = func() string {
	return ""
}

// SetSharedBinaryAccessor allows the integration package to register its binary accessor.
func SetSharedBinaryAccessor(accessor func() string) {
	getSharedBinaryPath = accessor
}

// GetBinaryPath returns the path to the compiled spinner binary.
// If TestMain has built a shared binary, it returns that path.
// Otherwise, it falls back to the default BinaryPath.
func GetBinaryPath() string {
	// Check if we have a shared binary from TestMain
	if sharedBinary := getSharedBinaryPath(); sharedBinary != "" {
		return sharedBinary
	}

	// Fall back to default path (for unit tests or non-integration tests)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		return BinaryPath
	}

	return filepath.Join(projectRoot, "dist", "spinner")
}

// RunCommand executes the spinner CLI with given arguments and returns output
func RunCommand(t *testing.T, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	return RunCommandWithEnv(t, nil, args...)
}

// RunCommandWithEnv executes the spinner CLI with custom environment variables
func RunCommandWithEnv(t *testing.T, env map[string]string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	binaryPath := GetBinaryPath()
	binaryPath, err := filepath.Abs(binaryPath)
	require.NoError(t, err, "failed to resolve binary path")

	cmd := exec.Command(binaryPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Start with current environment, but filter out SPINNER_* vars
	// to ensure tests run in isolation unless explicitly set
	var filteredEnv []string

	for _, e := range os.Environ() {
		// Skip SPINNER_* variables unless they're being explicitly set
		if len(e) >= 8 && e[:8] == "SPINNER_" {
			continue
		}

		filteredEnv = append(filteredEnv, e)
	}

	cmd.Env = filteredEnv

	// Add custom environment variables (including explicit SPINNER_* vars)
	for key, value := range env {
		if value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

	err = cmd.Run()

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()
	exitCode = 0

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	return stdout, stderr, exitCode
}

// RunCommandExpectSuccess runs a command and requires it to succeed
func RunCommandExpectSuccess(t *testing.T, args ...string) (stdout string, stderr string) {
	t.Helper()
	stdout, stderr, exitCode := RunCommand(t, args...)
	require.Equal(t, 0, exitCode, "command failed: stdout=%s, stderr=%s", stdout, stderr)

	return stdout, stderr
}

// RunCommandExpectError runs a command and requires it to fail
func RunCommandExpectError(t *testing.T, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	stdout, stderr, exitCode = RunCommand(t, args...)
	require.NotEqual(t, 0, exitCode, "command should have failed but succeeded: stdout=%s", stdout)

	return stdout, stderr, exitCode
}
