package integration

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

const (
	// Public test repository
	testRepo = "https://github.com/octocat/Hello-World.git"
	// Container workspace path
	workspacePath = "/home/spinner/workspace"
)

// TestSpin_SuccessfulContainerCreation tests successful container creation with valid flags
func TestSpin_SuccessfulContainerCreation(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, stdout, stderr := testutil.RunSpinCommand(t, args...)
	output := stdout + stderr

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Verify success message in output
	assert.Contains(t, output, "Instance created successfully", "should show success message")

	// Verify container was created
	assert.True(t, testutil.DockerContainerExists(t, containerName), "container should exist")
}

// TestSpin_ContainerNaming tests that container is named deterministically based on image + repo
func TestSpin_ContainerNaming(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Expected deterministic name format: spinner-<image-tag>-<repo-name>
	// For image "spinner:test-env" and repo "Hello-World", expect "spinner-test-env-hello-world"
	expectedName := "spinner-test-env-hello-world"

	assert.Equal(t, expectedName, containerName, "container name should be deterministic based on image and repo")
}

// TestSpin_ContainerRunning tests that container is running after spin command completes
func TestSpin_ContainerRunning(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Verify container is running
	assert.True(t, testutil.DockerContainerRunning(t, containerName), "container should be running")

	// Verify container status is "running"
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName)
	output, err := cmd.Output()
	require.NoError(t, err, "should get container status")

	status := strings.TrimSpace(string(output))
	assert.Equal(t, "running", status, "container status should be 'running'")
}

// TestSpin_RepositoryCloned tests that repository is cloned into /home/spinner/workspace inside container
func TestSpin_RepositoryCloned(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Wait a moment for clone to complete
	time.Sleep(2 * time.Second)

	// Check if /home/spinner/workspace exists and has .git directory
	cmd := exec.Command("docker", "exec", containerName, "test", "-d", workspacePath+"/.git")
	err := cmd.Run()
	assert.NoError(t, err, "repository should be cloned into %s", workspacePath)

	// Also verify the workspace directory exists
	cmd = exec.Command("docker", "exec", containerName, "test", "-d", workspacePath)
	err = cmd.Run()
	assert.NoError(t, err, "workspace directory should exist")
}

// TestSpin_ContainerExec tests that container can be exec'd into with bash
func TestSpin_ContainerExec(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Try to exec into container with bash
	cmd := exec.Command("docker", "exec", containerName, "bash", "-c", "pwd")
	output, err := cmd.Output()
	require.NoError(t, err, "should be able to exec into container with bash")

	// Verify we got output (pwd should return the current directory)
	assert.NotEmpty(t, output, "exec command should produce output")
	assert.Contains(t, string(output), "/", "pwd should return a valid path")
}

// TestSpin_NonExistentImage tests that non-existent Docker image exits with error
func TestSpin_NonExistentImage(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Use a non-existent image name
	nonExistentImage := "non-existent-image:latest"

	// Run spin command and expect it to fail
	args := []string{"spin", "--image", nonExistentImage, "--repo", testRepo}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	// Verify error message
	assert.Contains(t, output, "image '"+nonExistentImage+"' not found", "should show image not found error")

	// Verify exit code is 1
	assert.Equal(t, 1, exitCode, "should exit with code 1")
}

// TestSpin_PromptWithoutBranch tests that --prompt without --branch runs on default branch
func TestSpin_PromptWithoutBranch(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Run spin command with --prompt but without --branch
	args := []string{"spin", "--image", imageName, "--repo", testRepo, "--prompt", "echo test"}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Check environment variables in the container
	cmd := exec.Command("docker", "inspect", containerName, "-f", "{{range .Config.Env}}{{println .}}{{end}}")
	output, err := cmd.Output()
	require.NoError(t, err, "should get container environment")

	envVars := string(output)

	// Verify PROMPT is set
	assert.Contains(t, envVars, "PROMPT=", "PROMPT environment variable should be set")

	// Verify BRANCH is not set (or is empty)
	lines := strings.Split(envVars, "\n")
	branchSet := false

	for _, line := range lines {
		if strings.HasPrefix(line, "BRANCH=") && strings.TrimPrefix(line, "BRANCH=") != "" {
			branchSet = true
			break
		}
	}

	assert.False(t, branchSet, "BRANCH environment variable should not be set when not provided")
}

// TestSpin_BranchWithoutPrompt tests that --branch without --prompt creates idle container
func TestSpin_BranchWithoutPrompt(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Run spin command with --branch but without --prompt
	args := []string{"spin", "--image", imageName, "--repo", testRepo, "--branch", "test"}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Check environment variables in the container
	cmd := exec.Command("docker", "inspect", containerName, "-f", "{{range .Config.Env}}{{println .}}{{end}}")
	output, err := cmd.Output()
	require.NoError(t, err, "should get container environment")

	envVars := string(output)

	// Verify PROMPT is not set (container should be idle)
	lines := strings.Split(envVars, "\n")
	promptSet := false

	for _, line := range lines {
		if strings.HasPrefix(line, "PROMPT=") && strings.TrimPrefix(line, "PROMPT=") != "" {
			promptSet = true
			break
		}
	}

	assert.False(t, promptSet, "PROMPT environment variable should not be set (idle mode)")
}

// TestSpin_ReuseRunningContainer tests that running container is reused
func TestSpin_ReuseRunningContainer(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// First run: create container
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Verify container is running
	assert.True(t, testutil.DockerContainerRunning(t, containerName), "container should be running")

	// Second run: should reuse existing running container
	stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Verify output indicates reuse
	assert.Contains(t, output, "Reusing running instance: "+containerName, "should reuse running container")
}

// TestSpin_RestartStoppedContainer tests that stopped container is restarted
func TestSpin_RestartStoppedContainer(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// First run: create container
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Stop the container
	cmd := exec.Command("docker", "stop", containerName)
	err := cmd.Run()
	require.NoError(t, err, "should stop container")

	// Verify container is stopped
	assert.False(t, testutil.DockerContainerRunning(t, containerName), "container should be stopped")

	// Second run: should restart stopped container
	stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Verify output indicates restart
	assert.Contains(t, output, "Instance restarted: "+containerName, "should restart stopped container")

	// Verify container is running again
	assert.True(t, testutil.DockerContainerRunning(t, containerName), "container should be running after restart")
}

// TestSpin_PrivateRepoClone tests that private repository can be cloned with GITHUB_TOKEN
func TestSpin_PrivateRepoClone(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Use a private repository (assumes GITHUB_TOKEN is set)
	privateRepo := "https://github.com/rickihastings/spinner.git"

	// Run spin command with private repo
	args := []string{"spin", "--image", imageName, "--repo", privateRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Wait a moment for clone to complete
	time.Sleep(3 * time.Second)

	// Check if repository was cloned
	cmd := exec.Command("docker", "exec", containerName, "test", "-d", workspacePath+"/.git")
	err := cmd.Run()
	assert.NoError(t, err, "private repository should be cloned into %s", workspacePath)
}

// TestSpin_DeterministicNamingWithBranch tests container naming with branch
func TestSpin_DeterministicNamingWithBranch(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag
	testBranch := "master"

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// Run spin command with branch
	args := []string{"spin", "--image", imageName, "--repo", testRepo, "--prompt", "test", "--branch", testBranch}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Expected deterministic name format: spinner-<image-tag>-<repo-name>-<branch>
	// For image "spinner:test-env", repo "Hello-World", and branch "master"
	expectedName := "spinner-test-env-hello-world-master"

	assert.Equal(t, expectedName, containerName, "container name should include branch in deterministic naming")
}

// TestSpin_NameSanitization tests that container names are properly sanitized
func TestSpin_NameSanitization(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// Use SSH format repo URL with special chars that need sanitization
	sshRepo := "git@github.com:octocat/Hello-World.git"
	testBranch := "feature/auth-v2"

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", sshRepo, "--prompt", "test", "--branch", testBranch}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Expected: special characters should be replaced with hyphens
	// spinner-test-env-hello-world-feature-auth-v2
	expectedName := "spinner-test-env-hello-world-feature-auth-v2"

	assert.Equal(t, expectedName, containerName, "container name should be properly sanitized")

	// Verify name contains only valid Docker container name characters
	assert.Regexp(t, "^[a-z0-9][a-z0-9_.-]*$", containerName, "container name should only contain valid characters")
}

// TestSpin_RecreateFlag tests that --recreate flag removes and recreates container
func TestSpin_RecreateFlag(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// First run: create container
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Get container ID before recreate
	cmd := exec.Command("docker", "inspect", "-f", "{{.Id}}", containerName)
	output, err := cmd.Output()
	require.NoError(t, err, "should get container ID")

	containerIDBefore := strings.TrimSpace(string(output))

	// Second run: use --recreate flag to force recreation
	argsWithRecreate := []string{"spin", "--image", imageName, "--repo", testRepo, "--recreate"}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, argsWithRecreate...)
	recreateOutput := stdout + stderr

	// Get container ID after recreate
	cmd = exec.Command("docker", "inspect", "-f", "{{.Id}}", containerName)
	output, err = cmd.Output()
	require.NoError(t, err, "should get container ID after recreate")

	containerIDAfter := strings.TrimSpace(string(output))

	// Verify container was recreated (different container ID)
	assert.NotEqual(t, containerIDBefore, containerIDAfter, "container should have different ID after recreate")

	// Verify output indicates creation (not reuse)
	assert.Contains(t, recreateOutput, "Instance created successfully: "+containerName, "should show container creation message")
}

// TestSpin_SetupWithBaseImage tests that --setup with --base-image builds and creates container
func TestSpin_SetupWithBaseImage(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	imageTag := "setup-test"
	imageName := "spinner:" + imageTag

	// Cleanup
	t.Cleanup(func() {
		// Clean up container first (extract from potential error output)
		containerName := "spinner-" + imageTag + "-hello-world"
		testutil.RemoveDockerContainer(t, containerName)
		testutil.RemoveDockerImage(t, imageName)
	})

	// Run spin command with --setup and --base-image
	args := []string{"spin", "--setup", "--image", imageTag, "--base-image", "ubuntu:22.04", "--repo", testRepo, "--prompt", "echo test"}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Verify image build was successful
	assert.Contains(t, output, "Environment provisioned", "should show image build success message")

	// Verify container creation was successful
	assert.Contains(t, output, "Instance created successfully", "should show container creation message")

	// Verify the image was created
	assert.True(t, testutil.DockerImageExists(t, imageName), "image should exist after setup")

	// Extract container name and verify it's running
	var containerName string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Instance created successfully:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				containerName = parts[len(parts)-1]
				break
			}
		}
	}

	require.NotEmpty(t, containerName, "should extract container name")
	assert.True(t, testutil.DockerContainerRunning(t, containerName), "container should be running")
}

// TestSpin_SetupWithDockerfile tests that --setup with --dockerfile uses custom Dockerfile
func TestSpin_SetupWithDockerfile(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	imageTag := "dockerfile-test"
	imageName := "spinner:" + imageTag

	// Create a temporary custom Dockerfile
	tmpDir := t.TempDir()
	dockerfilePath := tmpDir + "/custom.Dockerfile"

	dockerfileContent := `FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
`
	err := testutil.WriteFile(t, dockerfilePath, dockerfileContent)
	require.NoError(t, err, "should create custom Dockerfile")

	// Cleanup
	t.Cleanup(func() {
		// Clean up container first
		containerName := "spinner-" + imageTag + "-hello-world"
		testutil.RemoveDockerContainer(t, containerName)
		testutil.RemoveDockerImage(t, imageName)
	})

	// Run spin command with --setup and --dockerfile
	args := []string{"spin", "--setup", "--image", imageTag, "--dockerfile", dockerfilePath, "--repo", testRepo, "--prompt", "echo test"}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Verify image build was successful
	assert.Contains(t, output, "Environment provisioned", "should show image build success message")

	// Verify container creation was successful
	assert.Contains(t, output, "Instance created successfully", "should show container creation message")

	// Verify the image was created
	assert.True(t, testutil.DockerImageExists(t, imageName), "image should exist after setup")

	// Extract container name and verify it's running
	var containerName string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Instance created successfully:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				containerName = parts[len(parts)-1]
				break
			}
		}
	}

	require.NotEmpty(t, containerName, "should extract container name")
	assert.True(t, testutil.DockerContainerRunning(t, containerName), "container should be running")
}

// TestSpin_EnvFilePassedToContainer tests that --env-file passes env vars into the container
// and copies the file to the workspace
func TestSpin_EnvFilePassedToContainer(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	_, imageName := testutil.SetupTestImage(t)

	// Create a temporary env file
	tmpDir := t.TempDir()
	envFilePath := tmpDir + "/.env"
	err := testutil.WriteFile(t, envFilePath, "NPM_TOKEN=abc123\nAPI_KEY=secret456\n")
	require.NoError(t, err, "should create env file")

	args := []string{"spin", "--image", imageName, "--repo", testRepo, "--env-file", envFilePath}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Verify env vars from the file are set in the container
	cmd := exec.Command("docker", "inspect", containerName, "-f", "{{range .Config.Env}}{{println .}}{{end}}")
	output, err := cmd.Output()
	require.NoError(t, err, "should get container environment")

	envVars := string(output)
	assert.Contains(t, envVars, "NPM_TOKEN=abc123", "env var from file should be set")
	assert.Contains(t, envVars, "API_KEY=secret456", "env var from file should be set")

	// Wait for startup to copy the file
	time.Sleep(2 * time.Second)

	// Verify the .env file was copied into the workspace
	cmd = exec.Command("docker", "exec", containerName, "test", "-f", workspacePath+"/.env")
	err = cmd.Run()
	assert.NoError(t, err, ".env file should exist in workspace")

	// Verify file contents match
	cmd = exec.Command("docker", "exec", containerName, "cat", workspacePath+"/.env")
	fileContent, err := cmd.Output()
	require.NoError(t, err, "should read .env file from workspace")
	assert.Contains(t, string(fileContent), "NPM_TOKEN=abc123", "workspace .env should contain env vars")
	assert.Contains(t, string(fileContent), "API_KEY=secret456", "workspace .env should contain env vars")
}

// TestSpin_EnvFileNotFoundError tests that --env-file with a non-existent file produces an error
func TestSpin_EnvFileNotFoundError(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	args := []string{"spin", "--image", "spinner:fake", "--repo", testRepo, "--env-file", "/nonexistent/path/.env"}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.NotEqual(t, 0, exitCode, "should fail with non-existent env file")
	assert.Contains(t, output, "--env-file: file does not exist", "should mention file not found")
}

// TestSpin_EnvFileWithEnvVarsCombined tests that --env-file and --env flags work together
func TestSpin_EnvFileWithEnvVarsCombined(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	_, imageName := testutil.SetupTestImage(t)

	// Create a temporary env file
	tmpDir := t.TempDir()
	envFilePath := tmpDir + "/.env"
	err := testutil.WriteFile(t, envFilePath, "FILE_VAR=from_file\n")
	require.NoError(t, err, "should create env file")

	args := []string{
		"spin", "--image", imageName, "--repo", testRepo,
		"--env-file", envFilePath,
		"--env", "CLI_VAR=from_cli",
	}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Verify both sources of env vars are set
	cmd := exec.Command("docker", "inspect", containerName, "-f", "{{range .Config.Env}}{{println .}}{{end}}")
	output, err := cmd.Output()
	require.NoError(t, err, "should get container environment")

	envVars := string(output)
	assert.Contains(t, envVars, "FILE_VAR=from_file", "env var from file should be set")
	assert.Contains(t, envVars, "CLI_VAR=from_cli", "env var from --env flag should be set")
}

// TestSpin_EnvVarsSingleVar tests that --env KEY=VALUE sets the env var in the container
func TestSpin_EnvVarsSingleVar(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	_, imageName := testutil.SetupTestImage(t)

	args := []string{"spin", "--image", imageName, "--repo", testRepo, "--env", "MY_CUSTOM_VAR=hello_world"}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Inspect container env vars
	cmd := exec.Command("docker", "inspect", containerName, "-f", "{{range .Config.Env}}{{println .}}{{end}}")
	output, err := cmd.Output()
	require.NoError(t, err, "should get container environment")

	assert.Contains(t, string(output), "MY_CUSTOM_VAR=hello_world", "custom env var should be set in container")
}

// TestSpin_EnvVarsMultipleVars tests that multiple --env flags all set their env vars
func TestSpin_EnvVarsMultipleVars(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	_, imageName := testutil.SetupTestImage(t)

	args := []string{
		"spin", "--image", imageName, "--repo", testRepo,
		"--env", "VAR_ONE=alpha",
		"--env", "VAR_TWO=beta",
		"--env", "VAR_THREE=gamma",
	}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	cmd := exec.Command("docker", "inspect", containerName, "-f", "{{range .Config.Env}}{{println .}}{{end}}")
	output, err := cmd.Output()
	require.NoError(t, err, "should get container environment")

	envVars := string(output)
	assert.Contains(t, envVars, "VAR_ONE=alpha", "first env var should be set")
	assert.Contains(t, envVars, "VAR_TWO=beta", "second env var should be set")
	assert.Contains(t, envVars, "VAR_THREE=gamma", "third env var should be set")
}

// TestSpin_EnvVarsReservedRejected tests that reserved variable names are rejected
func TestSpin_EnvVarsReservedRejected(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	reservedVars := []string{
		"GITHUB_TOKEN=fake",
		"CLAUDE_CODE_OAUTH_TOKEN=fake",
		"REPO_URL=fake",
		"PROMPT=fake",
		"BRANCH=fake",
		"MAX_ITERATIONS=fake",
	}

	for _, env := range reservedVars {
		key := strings.SplitN(env, "=", 2)[0]
		t.Run(key, func(t *testing.T) {
			args := []string{"spin", "--image", "spinner:fake", "--repo", testRepo, "--env", env}
			stdout, stderr, exitCode := testutil.RunCommandExpectError(t, args...)
			output := stdout + stderr

			assert.NotEqual(t, 0, exitCode, "should reject reserved variable %s", key)
			assert.Contains(t, output, "cannot override reserved variable", "should mention reserved variable")
		})
	}
}

// TestSpin_EnvVarsInvalidFormat tests that invalid --env format is rejected
func TestSpin_EnvVarsInvalidFormat(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	args := []string{"spin", "--image", "spinner:fake", "--repo", testRepo, "--env", "NO_EQUALS_SIGN"}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.NotEqual(t, 0, exitCode, "should reject invalid format")
	assert.Contains(t, output, "invalid format", "should mention invalid format")
}

// TestSpin_EnvVarsValueWithEquals tests that env values containing '=' are handled correctly
func TestSpin_EnvVarsValueWithEquals(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	_, imageName := testutil.SetupTestImage(t)

	args := []string{"spin", "--image", imageName, "--repo", testRepo, "--env", "DATABASE_URL=postgres://user:pass@host/db?sslmode=require"}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	cmd := exec.Command("docker", "inspect", containerName, "-f", "{{range .Config.Env}}{{println .}}{{end}}")
	output, err := cmd.Output()
	require.NoError(t, err, "should get container environment")

	assert.Contains(t, string(output), "DATABASE_URL=postgres://user:pass@host/db?sslmode=require",
		"env value containing '=' should be preserved correctly")
}

// TestSpin_SetupRebuildsExistingImage tests that --setup rebuilds image even if it exists
func TestSpin_SetupRebuildsExistingImage(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	imageTag := "rebuild-test"
	imageName := "spinner:" + imageTag

	// Cleanup
	t.Cleanup(func() {
		containerName := "spinner-" + imageTag + "-hello-world"
		testutil.RemoveDockerContainer(t, containerName)
		testutil.RemoveDockerImage(t, imageName)
	})

	// First, create the image using setup command
	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag, "--base-image", "ubuntu:22.04")

	// Verify the image exists before running spin with --setup
	assert.True(t, testutil.DockerImageExists(t, imageName), "initial image should exist")

	// Now run spin with --setup on the existing image
	args := []string{"spin", "--setup", "--image", imageTag, "--base-image", "ubuntu:22.04", "--repo", testRepo, "--prompt", "echo test"}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Verify image build was successful (this proves --setup triggered the build process)
	assert.Contains(t, output, "Environment provisioned", "should rebuild image even if it exists")

	// Verify container creation was successful
	assert.Contains(t, output, "Instance created successfully", "should show container creation message")

	// Verify the image still exists
	assert.True(t, testutil.DockerImageExists(t, imageName), "image should exist after rebuild")

	// Extract container name and verify it's running
	var containerName string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Instance created successfully:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				containerName = parts[len(parts)-1]
				break
			}
		}
	}

	require.NotEmpty(t, containerName, "should extract container name")
	assert.True(t, testutil.DockerContainerRunning(t, containerName), "container should be running")
}
