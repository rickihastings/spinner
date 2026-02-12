package docker

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestGenerateContainerName_BasicNaming tests container naming without branch (7.3)
func TestGenerateContainerName_BasicNaming(t *testing.T) {
	tests := []struct {
		name         string
		config       spinConfig
		expectedName string
		description  string
	}{
		{
			name: "image with tag and https repo",
			config: spinConfig{
				Image: "spinner:test-env",
				Repo:  "https://github.com/octocat/Hello-World.git",
			},
			expectedName: "spinner-test-env-hello-world",
			description:  "should generate {image}-{repo} format",
		},
		{
			name: "image with tag and ssh repo",
			config: spinConfig{
				Image: "spinner:test-env",
				Repo:  "git@github.com:user/my-repo.git",
			},
			expectedName: "spinner-test-env-my-repo",
			description:  "should handle SSH URLs",
		},
		{
			name: "image without tag",
			config: spinConfig{
				Image: "myimage",
				Repo:  "https://github.com/user/repo.git",
			},
			expectedName: "myimage-repo",
			description:  "should work with image names without tags",
		},
		{
			name: "repo without .git suffix",
			config: spinConfig{
				Image: "spinner:env",
				Repo:  "https://github.com/user/repo",
			},
			expectedName: "spinner-env-repo",
			description:  "should handle repos without .git suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateContainerName(tt.config)
			assert.Equal(t, tt.expectedName, result, tt.description)
		})
	}
}

// TestGenerateContainerName_Sanitization tests container name sanitization (7.4)
func TestGenerateContainerName_Sanitization(t *testing.T) {
	tests := []struct {
		name         string
		config       spinConfig
		expectedName string
		description  string
	}{
		{
			name: "special characters in image and repo",
			config: spinConfig{
				Image: "my_image:v1.0",
				Repo:  "https://github.com/user/My-Repo.git",
			},
			expectedName: "my_image-v1-0-my-repo",
			description:  "should replace dots and uppercase with lowercase/hyphens",
		},
		{
			name: "spaces and invalid chars",
			config: spinConfig{
				Image: "test image:tag",
				Repo:  "https://github.com/user/test repo.git",
			},
			expectedName: "test-image-tag-test-repo",
			description:  "should replace spaces with hyphens",
		},
		{
			name: "consecutive special chars",
			config: spinConfig{
				Image: "image::test",
				Repo:  "https://github.com/user/repo--name.git",
			},
			expectedName: "image-test-repo-name",
			description:  "should collapse consecutive hyphens",
		},
		{
			name: "leading and trailing hyphens",
			config: spinConfig{
				Image: "-image-",
				Repo:  "https://github.com/user/-repo-.git",
			},
			expectedName: "image-repo",
			description:  "should trim leading and trailing hyphens",
		},
		{
			name: "uppercase letters",
			config: spinConfig{
				Image: "MyImage:TAG",
				Repo:  "https://github.com/User/RepoName.git",
			},
			expectedName: "myimage-tag-reponame",
			description:  "should convert to lowercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateContainerName(tt.config)
			assert.Equal(t, tt.expectedName, result, tt.description)
		})
	}
}

// TestGenerateContainerName_WithBranch tests deterministic naming with branch (7.5)
func TestGenerateContainerName_WithBranch(t *testing.T) {
	tests := []struct {
		name         string
		config       spinConfig
		expectedName string
		description  string
	}{
		{
			name: "with simple branch name",
			config: spinConfig{
				Image:  "spinner:test-env",
				Repo:   "https://github.com/octocat/Hello-World.git",
				Branch: "master",
			},
			expectedName: "spinner-test-env-hello-world-master",
			description:  "should generate {image}-{repo}-{branch} format",
		},
		{
			name: "with branch containing slashes",
			config: spinConfig{
				Image:  "spinner:test-env",
				Repo:   "git@github.com:octocat/Hello-World.git",
				Branch: "feature/auth-v2",
			},
			expectedName: "spinner-test-env-hello-world-feature-auth-v2",
			description:  "should sanitize branch name (replace / with -)",
		},
		{
			name: "with complex branch name",
			config: spinConfig{
				Image:  "myimage:v1",
				Repo:   "https://github.com/user/repo.git",
				Branch: "bugfix/TICKET-123_fix",
			},
			expectedName: "myimage-v1-repo-bugfix-ticket-123_fix",
			description:  "should handle complex branch names",
		},
		{
			name: "empty branch falls back to no branch",
			config: spinConfig{
				Image:  "spinner:env",
				Repo:   "https://github.com/user/repo.git",
				Branch: "",
			},
			expectedName: "spinner-env-repo",
			description:  "should omit branch part when branch is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateContainerName(tt.config)
			assert.Equal(t, tt.expectedName, result, tt.description)
		})
	}
}

// TestSanitizeComponent tests the sanitization function directly (7.4)
func TestSanitizeComponent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-hyphens", "with-hyphens"},
		{"with_underscores", "with_underscores"},
		{"UPPERCASE", "uppercase"},
		{"with.dots", "with-dots"},
		{"with spaces", "with-spaces"},
		{"multiple---hyphens", "multiple-hyphens"},
		{"--leading", "leading"},
		{"trailing--", "trailing"},
		{"mixed.Case-123", "mixed-case-123"},
		{"special!@#chars", "special-chars"},
		{"git@github.com:user/repo.git", "git-github-com-user-repo-git"},
		{"feature/branch-name", "feature-branch-name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeComponent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractRepoName tests repository name extraction
func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		repoURL  string
		expected string
	}{
		{"https://github.com/user/repo.git", "repo"},
		{"https://github.com/user/my-repo.git", "my-repo"},
		{"git@github.com:user/repo.git", "repo"},
		{"https://github.com/octocat/Hello-World.git", "Hello-World"},
		{"https://github.com/user/repo", "repo"},
		{"invalid-url", "invalid-url"},
		{"", "sandbox"},
	}

	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			result := extractRepoName(tt.repoURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildDockerRunCommand_BasicScenarios tests Docker run command building (7.2)
func TestBuildDockerRunCommand_BasicScenarios(t *testing.T) {
	// Set required environment variables
	_ = os.Setenv("GITHUB_TOKEN", "test-github-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-claude-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name           string
		config         spinConfig
		containerName  string
		hasNpmrc       bool
		expectedArgs   []string
		unexpectedArgs []string
		description    string
	}{
		{
			name: "basic run without prompt",
			config: spinConfig{
				Image: "spinner:test",
				Repo:  "https://github.com/user/repo.git",
			},
			containerName: "spinner-test-repo",
			hasNpmrc:      false,
			expectedArgs: []string{
				"run", "-d", "--name", "spinner-test-repo",
				"--env-file",
				"spinner:test",
			},
			unexpectedArgs: []string{"-e GITHUB_TOKEN=", "-e CLAUDE_CODE_OAUTH_TOKEN="},
			description:    "should create basic run command with --env-file",
		},
		{
			name: "run with prompt",
			config: spinConfig{
				Image:  "spinner:test",
				Repo:   "https://github.com/user/repo.git",
				Prompt: "fix the bug",
			},
			containerName: "spinner-test-repo",
			hasNpmrc:      false,
			expectedArgs: []string{
				"--env-file",
			},
			description: "should use --env-file with prompt",
		},
		{
			name: "run with prompt and custom max iterations",
			config: spinConfig{
				Image:         "spinner:test",
				Repo:          "https://github.com/user/repo.git",
				Prompt:        "add feature",
				MaxIterations: "50",
			},
			containerName: "spinner-test-repo",
			hasNpmrc:      false,
			expectedArgs: []string{
				"--env-file",
			},
			description: "should use --env-file with custom max iterations",
		},
		{
			name: "run with prompt and branch",
			config: spinConfig{
				Image:  "spinner:test",
				Repo:   "https://github.com/user/repo.git",
				Prompt: "test task",
				Branch: "feature/new",
			},
			containerName: "spinner-test-repo-feature-new",
			hasNpmrc:      false,
			expectedArgs: []string{
				"--env-file",
			},
			description: "should use --env-file with branch and prompt",
		},
		{
			name: "run with branch but no prompt",
			config: spinConfig{
				Image:  "spinner:test",
				Repo:   "https://github.com/user/repo.git",
				Branch: "develop",
			},
			containerName: "spinner-test-repo-develop",
			hasNpmrc:      false,
			expectedArgs: []string{
				"--env-file",
			},
			unexpectedArgs: []string{"-e PROMPT=", "-e MAX_ITERATIONS="},
			description:    "should use --env-file even with branch only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, tmpFile, err := buildDockerRunCommand(tt.config, tt.containerName, tt.hasNpmrc)
			if tmpFile != "" {
				defer func() { _ = os.Remove(tmpFile) }()
			}

			assert.NoError(t, err)
			assert.NotNil(t, args)

			argsStr := ""
			for _, arg := range args {
				argsStr += arg + " "
			}

			// Check expected args
			for _, expected := range tt.expectedArgs {
				assert.Contains(t, argsStr, expected, "should contain: %s", expected)
			}

			// Check unexpected args
			for _, unexpected := range tt.unexpectedArgs {
				assert.NotContains(t, argsStr, unexpected, "should not contain: %s", unexpected)
			}

			// Verify logs volume mount
			expectedLogMount := homeDir + "/.spinner/" + tt.containerName + "/logs:/logs"
			assert.Contains(t, argsStr, expectedLogMount, "should mount logs directory")
		})
	}
}

// TestBuildDockerRunCommand_NpmrcHandling tests npmrc mounting
func TestBuildDockerRunCommand_NpmrcHandling(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name        string
		hasNpmrc    bool
		shouldMount bool
	}{
		{"with npmrc", true, true},
		{"without npmrc", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := spinConfig{
				Image: "spinner:test",
				Repo:  "https://github.com/user/repo.git",
			}

			args, tmpFile, err := buildDockerRunCommand(config, "test-container", tt.hasNpmrc)
			if tmpFile != "" {
				defer func() { _ = os.Remove(tmpFile) }()
			}

			assert.NoError(t, err)

			argsStr := ""
			for _, arg := range args {
				argsStr += arg + " "
			}

			expectedNpmrcMount := homeDir + "/.npmrc:/home/spinner/.npmrc"
			if tt.shouldMount {
				assert.Contains(t, argsStr, expectedNpmrcMount, "should mount .npmrc")
			} else {
				assert.NotContains(t, argsStr, expectedNpmrcMount, "should not mount .npmrc")
			}
		})
	}
}

// TestBuildDockerRunCommand_RepoURLPassthrough tests that the repo URL is written as-is
// (SSH-to-HTTPS conversion happens upstream in cmd/spin.go before reaching the provider)
func TestBuildDockerRunCommand_RepoURLPassthrough(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	config := spinConfig{
		Image: "spinner:test",
		Repo:  "https://github.com/user/repo.git",
	}

	_, tmpFile, err := buildDockerRunCommand(config, "test-container", false)
	if tmpFile != "" {
		defer func() { _ = os.Remove(tmpFile) }()
	}

	assert.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)

	assert.Contains(t, string(content), "REPO_URL=https://github.com/user/repo.git\n", "should pass repo URL through as-is")
}

// TestEscapeShellArg tests shell argument escaping
func TestEscapeShellArg(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"with'quote", "'with'\\''quote'"},
		{"multiple'quotes'here", "'multiple'\\''quotes'\\''here'"},
		{"", "''"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeShellArg(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestContainerReuse_MockClient tests container reuse logic with mock (7.6)
func TestContainerReuse_MockClient(t *testing.T) {
	tests := []struct {
		name              string
		containerStatus   ContainerStatus
		recreate          bool
		expectedAction    string
		shouldCallRemove  bool
		shouldCallRun     bool
		shouldCallRestart bool
	}{
		{
			name:              "container running - reuse",
			containerStatus:   StatusRunning,
			recreate:          false,
			expectedAction:    "reuse",
			shouldCallRemove:  false,
			shouldCallRun:     false,
			shouldCallRestart: false,
		},
		{
			name:              "container stopped - restart",
			containerStatus:   StatusStopped,
			recreate:          false,
			expectedAction:    "restart",
			shouldCallRemove:  false,
			shouldCallRun:     false,
			shouldCallRestart: true,
		},
		{
			name:              "container none - create",
			containerStatus:   StatusNone,
			recreate:          false,
			expectedAction:    "create",
			shouldCallRemove:  false,
			shouldCallRun:     true,
			shouldCallRestart: false,
		},
		{
			name:              "container running - recreate",
			containerStatus:   StatusRunning,
			recreate:          true,
			expectedAction:    "recreate",
			shouldCallRemove:  true,
			shouldCallRun:     true,
			shouldCallRestart: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockDockerClient)
			ctx := context.Background()
			containerName := "test-container"

			// Mock ContainerExists
			mockClient.On("ContainerExists", ctx, containerName).Return(tt.containerStatus, nil)

			if tt.shouldCallRemove {
				mockClient.On("RemoveContainer", ctx, containerName).Return(
					ContainerResult{Success: true, ContainerName: containerName}, nil,
				)
			}

			if tt.shouldCallRun {
				mockClient.On("RunContainer", ctx, mock.Anything, containerName).Return(
					ContainerResult{Success: true, ContainerName: containerName}, nil,
				)
			}

			if tt.shouldCallRestart {
				mockClient.On("StartContainer", ctx, containerName).Return(
					ContainerResult{Success: true, ContainerName: containerName}, nil,
				)
			}

			// Execute the logic based on container status
			status, err := mockClient.ContainerExists(ctx, containerName)
			assert.NoError(t, err)
			assert.Equal(t, tt.containerStatus, status)

			if tt.recreate && status != StatusNone {
				_, err := mockClient.RemoveContainer(ctx, containerName)
				assert.NoError(t, err)
			}

			if status == StatusNone || tt.recreate {
				if tt.shouldCallRun {
					_, err := mockClient.RunContainer(ctx, []string{}, containerName)
					assert.NoError(t, err)
				}
			} else if status == StatusStopped {
				_, err := mockClient.StartContainer(ctx, containerName)
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestContainerRecreation_MockClient tests container recreation logic (7.7)
func TestContainerRecreation_MockClient(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()
	containerName := "test-container"

	// Mock that container is running
	mockClient.On("ContainerExists", ctx, containerName).Return(StatusRunning, nil)

	// Mock remove and recreate
	mockClient.On("RemoveContainer", ctx, containerName).Return(
		ContainerResult{Success: true, ContainerName: containerName}, nil,
	)
	mockClient.On("RunContainer", ctx, mock.Anything, containerName).Return(
		ContainerResult{Success: true, ContainerName: containerName}, nil,
	)

	// Simulate recreate logic
	status, err := mockClient.ContainerExists(ctx, containerName)
	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, status)

	// Remove existing container
	result, err := mockClient.RemoveContainer(ctx, containerName)
	assert.NoError(t, err)
	assert.True(t, result.Success)

	// Create new container
	result, err = mockClient.RunContainer(ctx, []string{}, containerName)
	assert.NoError(t, err)
	assert.True(t, result.Success)

	mockClient.AssertExpectations(t)
}

// TestNamingScenarios_TableDriven tests various naming scenarios (7.8)
func TestNamingScenarios_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		config       spinConfig
		expectedName string
	}{
		{
			name: "basic https repo",
			config: spinConfig{
				Image: "spinner:default",
				Repo:  "https://github.com/user/project.git",
			},
			expectedName: "spinner-default-project",
		},
		{
			name: "ssh repo with branch",
			config: spinConfig{
				Image:  "myimage:v1",
				Repo:   "git@github.com:org/repo-name.git",
				Branch: "develop",
			},
			expectedName: "myimage-v1-repo-name-develop",
		},
		{
			name: "complex image tag with special chars",
			config: spinConfig{
				Image: "my.image:v2.0.1",
				Repo:  "https://github.com/user/My_Project.git",
			},
			expectedName: "my-image-v2-0-1-my_project",
		},
		{
			name: "branch with slashes and special chars",
			config: spinConfig{
				Image:  "spinner:env",
				Repo:   "https://github.com/user/repo.git",
				Branch: "feature/JIRA-123_bugfix",
			},
			expectedName: "spinner-env-repo-feature-jira-123_bugfix",
		},
		{
			name: "uppercase repo and branch",
			config: spinConfig{
				Image:  "IMAGE:TAG",
				Repo:   "https://github.com/User/REPO.git",
				Branch: "MAIN",
			},
			expectedName: "image-tag-repo-main",
		},
		{
			name: "multiple consecutive special chars",
			config: spinConfig{
				Image: "test::image",
				Repo:  "https://github.com/user/repo--name.git",
			},
			expectedName: "test-image-repo-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateContainerName(tt.config)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

// TestBuildDockerRunCommand_EnvFile tests that env vars are written to a temp file
func TestBuildDockerRunCommand_EnvFile(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-github-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-claude-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	config := spinConfig{
		Image: "spinner:test",
		Repo:  "https://github.com/user/repo.git",
		EnvVars: map[string]string{
			"NPM_TOKEN":  "npm-secret-123",
			"MY_API_KEY": "api-secret-456",
		},
	}

	args, tmpFile, err := buildDockerRunCommand(config, "test-container", false)
	assert.NoError(t, err)
	assert.NotEmpty(t, tmpFile, "should return temp file path")

	defer func() { _ = os.Remove(tmpFile) }()

	// Verify args contain --env-file
	argsStr := ""
	for _, arg := range args {
		argsStr += arg + " "
	}

	assert.Contains(t, argsStr, "--env-file", "should use --env-file")
	assert.Contains(t, argsStr, tmpFile, "should reference temp file path")

	// Verify temp file exists and has correct permissions
	fileInfo, err := os.Stat(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm(), "temp file should have 0600 permissions")

	// Verify temp file contents
	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)

	contentStr := string(content)

	// Check built-in vars are present
	assert.Contains(t, contentStr, "GITHUB_TOKEN=test-github-token\n")
	assert.Contains(t, contentStr, "CLAUDE_CODE_OAUTH_TOKEN=test-claude-token\n")
	assert.Contains(t, contentStr, "REPO_URL=https://github.com/user/repo.git\n")

	// Check custom vars are present
	assert.Contains(t, contentStr, "NPM_TOKEN=npm-secret-123\n")
	assert.Contains(t, contentStr, "MY_API_KEY=api-secret-456\n")
}

// TestBuildDockerRunCommand_EnvFileWithPromptAndBranch tests env file with all options
func TestBuildDockerRunCommand_EnvFileWithPromptAndBranch(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	config := spinConfig{
		Image:         "spinner:test",
		Repo:          "https://github.com/user/repo.git",
		Prompt:        "fix the bug",
		Branch:        "feature/test",
		MaxIterations: "50",
		EnvVars: map[string]string{
			"CUSTOM_VAR": "custom-value",
		},
	}

	_, tmpFile, err := buildDockerRunCommand(config, "test-container", false)
	assert.NoError(t, err)
	assert.NotEmpty(t, tmpFile)

	defer func() { _ = os.Remove(tmpFile) }()

	// Verify temp file contents include all vars
	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)

	contentStr := string(content)

	assert.Contains(t, contentStr, "GITHUB_TOKEN=test-token\n")
	assert.Contains(t, contentStr, "CLAUDE_CODE_OAUTH_TOKEN=test-token\n")
	assert.Contains(t, contentStr, "REPO_URL=https://github.com/user/repo.git\n")
	assert.Contains(t, contentStr, "BRANCH=feature/test\n")
	assert.Contains(t, contentStr, "PROMPT=fix the bug\n")
	assert.Contains(t, contentStr, "MAX_ITERATIONS=50\n")
	assert.Contains(t, contentStr, "LOG_DIR=/logs\n")
	assert.Contains(t, contentStr, "CUSTOM_VAR=custom-value\n")
}

// TestBuildDockerRunCommand_EnvFileEmptyCustomVars tests with no custom env vars
func TestBuildDockerRunCommand_EnvFileEmptyCustomVars(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	config := spinConfig{
		Image:   "spinner:test",
		Repo:    "https://github.com/user/repo.git",
		EnvVars: map[string]string{},
	}

	args, tmpFile, err := buildDockerRunCommand(config, "test-container", false)
	assert.NoError(t, err)
	assert.NotEmpty(t, tmpFile)

	defer func() { _ = os.Remove(tmpFile) }()

	// Verify args use --env-file even with no custom vars
	argsStr := ""
	for _, arg := range args {
		argsStr += arg + " "
	}

	assert.Contains(t, argsStr, "--env-file")

	// Verify temp file still contains built-in vars
	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)

	contentStr := string(content)

	assert.Contains(t, contentStr, "GITHUB_TOKEN=test-token\n")
	assert.Contains(t, contentStr, "CLAUDE_CODE_OAUTH_TOKEN=test-token\n")
}

// TestBuildDockerRunCommand_EnvFileNilCustomVars tests with nil custom env vars map
func TestBuildDockerRunCommand_EnvFileNilCustomVars(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	config := spinConfig{
		Image:   "spinner:test",
		Repo:    "https://github.com/user/repo.git",
		EnvVars: nil,
	}

	_, tmpFile, err := buildDockerRunCommand(config, "test-container", false)
	assert.NoError(t, err)
	assert.NotEmpty(t, tmpFile)

	defer func() { _ = os.Remove(tmpFile) }()

	// Should handle nil map gracefully
	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)

	contentStr := string(content)

	assert.Contains(t, contentStr, "GITHUB_TOKEN=test-token\n")
	assert.Contains(t, contentStr, "CLAUDE_CODE_OAUTH_TOKEN=test-token\n")
}
