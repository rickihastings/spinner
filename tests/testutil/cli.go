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

// BuildCLI compiles the spinner binary for testing
func BuildCLI(t *testing.T) string {
	t.Helper()

	// Get project root (two levels up from tests/testutil)
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err, "failed to get project root")

	// Build the binary
	outputPath := filepath.Join(projectRoot, "dist", "spinner")
	cmd := exec.Command("go", "build", "-o", outputPath)
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build CLI binary: %s", string(output))

	return outputPath
}

// RunCommand executes the spinner CLI with given arguments and returns output
func RunCommand(t *testing.T, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	return RunCommandWithEnv(t, nil, args...)
}

// RunCommandWithEnv executes the spinner CLI with custom environment variables
func RunCommandWithEnv(t *testing.T, env map[string]string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	binaryPath, err := filepath.Abs(BinaryPath)
	require.NoError(t, err, "failed to resolve binary path")

	cmd := exec.Command(binaryPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Start with current environment
	cmd.Env = os.Environ()

	// Add custom environment variables
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
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
