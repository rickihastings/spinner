package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExecCommand_Construction tests that the exec command is constructed correctly
func TestExecCommand_Construction(t *testing.T) {
	cmd := NewExecCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "exec", cmd.Use)
	assert.Contains(t, cmd.Short, "autonomous iteration loop")
	assert.NotNil(t, cmd.RunE)
}

// TestExecCommand_HelpText tests that help text is displayed correctly
func TestExecCommand_HelpText(t *testing.T) {
	cmd := NewExecCommand()

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()

	assert.NoError(t, err)

	output := b.String()
	assert.Contains(t, output, "Execute the autonomous iteration loop")
	assert.Contains(t, output, "ENVIRONMENT VARIABLES")
	assert.Contains(t, output, "PROMPT")
	assert.Contains(t, output, "MAX_ITERATIONS")
}

// TestExecCommand_MissingPrompt tests that the command fails when PROMPT is missing
func TestExecCommand_MissingPrompt(t *testing.T) {
	// Set MAX_ITERATIONS but not PROMPT
	_ = os.Setenv("MAX_ITERATIONS", "10")
	_ = os.Unsetenv("PROMPT")

	defer func() { _ = os.Unsetenv("MAX_ITERATIONS") }()

	cmd := NewExecCommand()

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.RunE(cmd, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PROMPT is required")
}

// TestExecCommand_MissingMaxIterations tests that the command fails when MAX_ITERATIONS is missing
func TestExecCommand_MissingMaxIterations(t *testing.T) {
	// Set PROMPT but not MAX_ITERATIONS
	_ = os.Setenv("PROMPT", "Test prompt")
	_ = os.Unsetenv("MAX_ITERATIONS")

	defer func() { _ = os.Unsetenv("PROMPT") }()

	cmd := NewExecCommand()

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.RunE(cmd, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MAX_ITERATIONS is required")
}

// TestExecCommand_InvalidMaxIterations tests that the command fails when MAX_ITERATIONS is invalid
func TestExecCommand_InvalidMaxIterations(t *testing.T) {
	tests := []struct {
		name         string
		maxIter      string
		errorMessage string
	}{
		{
			name:         "negative value",
			maxIter:      "-1",
			errorMessage: "MAX_ITERATIONS must be greater than 0",
		},
		{
			name:         "zero value",
			maxIter:      "0",
			errorMessage: "MAX_ITERATIONS is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("PROMPT", "Test prompt")
			_ = os.Setenv("MAX_ITERATIONS", tt.maxIter)

			defer func() {
				_ = os.Unsetenv("PROMPT")
				_ = os.Unsetenv("MAX_ITERATIONS")
			}()

			cmd := NewExecCommand()

			b := new(bytes.Buffer)
			cmd.SetOut(b)
			cmd.SetErr(b)
			cmd.SetArgs([]string{})

			err := cmd.RunE(cmd, []string{})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMessage)
		})
	}
}

// TestExecCommand_ValidConfig tests that valid configuration is accepted
func TestExecCommand_ValidConfig(t *testing.T) {
	// Skip this test as it would attempt to run the full loop
	// This is more of an integration test
	t.Skip("Skipping integration test - requires full loop execution")

	_ = os.Setenv("PROMPT", "Test prompt")
	_ = os.Setenv("MAX_ITERATIONS", "1")
	_ = os.Setenv("BRANCH", "test-branch")

	defer func() {
		_ = os.Unsetenv("PROMPT")
		_ = os.Unsetenv("MAX_ITERATIONS")
		_ = os.Unsetenv("BRANCH")
	}()

	cmd := NewExecCommand()

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	// This would actually run the loop, so we skip it
	_ = cmd
}
