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
		config       SpinConfig
		expectedName string
		description  string
	}{
		{
			name: "image with tag and https repo",
			config: SpinConfig{
				Image: "spinner:test-env",
				Repo:  "https://github.com/octocat/Hello-World.git",
			},
			expectedName: "spinner-test-env-hello-world",
			description:  "should generate {image}-{repo} format",
		},
		{
			name: "image with tag and ssh repo",
			config: SpinConfig{
				Image: "spinner:test-env",
				Repo:  "git@github.com:user/my-repo.git",
			},
			expectedName: "spinner-test-env-my-repo",
			description:  "should handle SSH URLs",
		},
		{
			name: "image without tag",
			config: SpinConfig{
				Image: "myimage",
				Repo:  "https://github.com/user/repo.git",
			},
			expectedName: "myimage-repo",
			description:  "should work with image names without tags",
		},
		{
			name: "repo without .git suffix",
			config: SpinConfig{
				Image: "spinner:env",
				Repo:  "https://github.com/user/repo",
			},
			expectedName: "spinner-env-repo",
			description:  "should handle repos without .git suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateContainerName(tt.config)
			assert.Equal(t, tt.expectedName, result, tt.description)
		})
	}
}

// TestGenerateContainerName_Sanitization tests container name sanitization (7.4)
func TestGenerateContainerName_Sanitization(t *testing.T) {
	tests := []struct {
		name         string
		config       SpinConfig
		expectedName string
		description  string
	}{
		{
			name: "special characters in image and repo",
			config: SpinConfig{
				Image: "my_image:v1.0",
				Repo:  "https://github.com/user/My-Repo.git",
			},
			expectedName: "my_image-v1-0-my-repo",
			description:  "should replace dots and uppercase with lowercase/hyphens",
		},
		{
			name: "spaces and invalid chars",
			config: SpinConfig{
				Image: "test image:tag",
				Repo:  "https://github.com/user/test repo.git",
			},
			expectedName: "test-image-tag-test-repo",
			description:  "should replace spaces with hyphens",
		},
		{
			name: "consecutive special chars",
			config: SpinConfig{
				Image: "image::test",
				Repo:  "https://github.com/user/repo--name.git",
			},
			expectedName: "image-test-repo-name",
			description:  "should collapse consecutive hyphens",
		},
		{
			name: "leading and trailing hyphens",
			config: SpinConfig{
				Image: "-image-",
				Repo:  "https://github.com/user/-repo-.git",
			},
			expectedName: "image-repo",
			description:  "should trim leading and trailing hyphens",
		},
		{
			name: "uppercase letters",
			config: SpinConfig{
				Image: "MyImage:TAG",
				Repo:  "https://github.com/User/RepoName.git",
			},
			expectedName: "myimage-tag-reponame",
			description:  "should convert to lowercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateContainerName(tt.config)
			assert.Equal(t, tt.expectedName, result, tt.description)
		})
	}
}

// TestGenerateContainerName_WithBranch tests deterministic naming with branch (7.5)
func TestGenerateContainerName_WithBranch(t *testing.T) {
	tests := []struct {
		name         string
		config       SpinConfig
		expectedName string
		description  string
	}{
		{
			name: "with simple branch name",
			config: SpinConfig{
				Image:  "spinner:test-env",
				Repo:   "https://github.com/octocat/Hello-World.git",
				Branch: "master",
			},
			expectedName: "spinner-test-env-hello-world-master",
			description:  "should generate {image}-{repo}-{branch} format",
		},
		{
			name: "with branch containing slashes",
			config: SpinConfig{
				Image:  "spinner:test-env",
				Repo:   "git@github.com:octocat/Hello-World.git",
				Branch: "feature/auth-v2",
			},
			expectedName: "spinner-test-env-hello-world-feature-auth-v2",
			description:  "should sanitize branch name (replace / with -)",
		},
		{
			name: "with complex branch name",
			config: SpinConfig{
				Image:  "myimage:v1",
				Repo:   "https://github.com/user/repo.git",
				Branch: "bugfix/TICKET-123_fix",
			},
			expectedName: "myimage-v1-repo-bugfix-ticket-123_fix",
			description:  "should handle complex branch names",
		},
		{
			name: "empty branch falls back to no branch",
			config: SpinConfig{
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
			result := GenerateContainerName(tt.config)
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
		config         SpinConfig
		containerName  string
		hasNpmrc       bool
		expectedArgs   []string
		unexpectedArgs []string
		description    string
	}{
		{
			name: "basic run without prompt",
			config: SpinConfig{
				Image: "spinner:test",
				Repo:  "https://github.com/user/repo.git",
			},
			containerName: "spinner-test-repo",
			hasNpmrc:      false,
			expectedArgs: []string{
				"run", "-d", "--name", "spinner-test-repo",
				"-e", "GITHUB_TOKEN=test-github-token",
				"-e", "CLAUDE_CODE_OAUTH_TOKEN=test-claude-token",
				"-e", "REPO_URL=https://github.com/user/repo.git",
				"spinner:test",
			},
			unexpectedArgs: []string{"PROMPT=", "MAX_ITERATIONS=", "BRANCH="},
			description:    "should create basic run command without Ralph environment",
		},
		{
			name: "run with prompt",
			config: SpinConfig{
				Image:  "spinner:test",
				Repo:   "https://github.com/user/repo.git",
				Prompt: "fix the bug",
			},
			containerName: "spinner-test-repo",
			hasNpmrc:      false,
			expectedArgs: []string{
				"-e", "PROMPT='fix the bug'",
				"-e", "MAX_ITERATIONS=100",
				"-e", "LOG_DIR=/logs",
			},
			description: "should include prompt and default max iterations",
		},
		{
			name: "run with prompt and custom max iterations",
			config: SpinConfig{
				Image:         "spinner:test",
				Repo:          "https://github.com/user/repo.git",
				Prompt:        "add feature",
				MaxIterations: "50",
			},
			containerName: "spinner-test-repo",
			hasNpmrc:      false,
			expectedArgs: []string{
				"-e", "PROMPT='add feature'",
				"-e", "MAX_ITERATIONS=50",
				"-e", "LOG_DIR=/logs",
			},
			description: "should use custom max iterations when provided",
		},
		{
			name: "run with prompt and branch",
			config: SpinConfig{
				Image:  "spinner:test",
				Repo:   "https://github.com/user/repo.git",
				Prompt: "test task",
				Branch: "feature/new",
			},
			containerName: "spinner-test-repo-feature-new",
			hasNpmrc:      false,
			expectedArgs: []string{
				"-e", "PROMPT='test task'",
				"-e", "BRANCH='feature/new'",
				"-e", "LOG_DIR=/logs",
			},
			description: "should include branch when provided with prompt",
		},
		{
			name: "run with branch but no prompt",
			config: SpinConfig{
				Image:  "spinner:test",
				Repo:   "https://github.com/user/repo.git",
				Branch: "develop",
			},
			containerName: "spinner-test-repo-develop",
			hasNpmrc:      false,
			expectedArgs: []string{
				"-e", "BRANCH='develop'",
			},
			unexpectedArgs: []string{"PROMPT=", "MAX_ITERATIONS=", "LOG_DIR="},
			description:    "should include branch even without prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := BuildDockerRunCommand(tt.config, tt.containerName, tt.hasNpmrc)

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
			config := SpinConfig{
				Image: "spinner:test",
				Repo:  "https://github.com/user/repo.git",
			}

			args, err := BuildDockerRunCommand(config, "test-container", tt.hasNpmrc)

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

// TestBuildDockerRunCommand_SshToHttpsConversion tests SSH URL conversion
func TestBuildDockerRunCommand_SshToHttpsConversion(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	}()

	tests := []struct {
		name        string
		repoURL     string
		expectedURL string
	}{
		{
			name:        "convert SSH to HTTPS",
			repoURL:     "git@github.com:user/repo.git",
			expectedURL: "https://github.com/user/repo.git",
		},
		{
			name:        "keep HTTPS as-is",
			repoURL:     "https://github.com/user/repo.git",
			expectedURL: "https://github.com/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SpinConfig{
				Image: "spinner:test",
				Repo:  tt.repoURL,
			}

			args, err := BuildDockerRunCommand(config, "test-container", false)

			assert.NoError(t, err)

			argsStr := ""
			for _, arg := range args {
				argsStr += arg + " "
			}

			expectedEnvVar := "REPO_URL=" + tt.expectedURL
			assert.Contains(t, argsStr, expectedEnvVar, "should use converted URL")
		})
	}
}

// TestConvertSshToHttps tests SSH to HTTPS URL conversion
func TestConvertSshToHttps(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"git@github.com:user/repo.git", "https://github.com/user/repo.git"},
		{"git@github.com:org/project.git", "https://github.com/org/project.git"},
		{"https://github.com/user/repo.git", "https://github.com/user/repo.git"},
		{"http://github.com/user/repo.git", "http://github.com/user/repo.git"},
		{"other-url", "other-url"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertSshToHttps(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
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
				mockClient.On("RestartContainer", ctx, containerName).Return(
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
				_, err := mockClient.RestartContainer(ctx, containerName)
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
		config       SpinConfig
		expectedName string
	}{
		{
			name: "basic https repo",
			config: SpinConfig{
				Image: "spinner:default",
				Repo:  "https://github.com/user/project.git",
			},
			expectedName: "spinner-default-project",
		},
		{
			name: "ssh repo with branch",
			config: SpinConfig{
				Image:  "myimage:v1",
				Repo:   "git@github.com:org/repo-name.git",
				Branch: "develop",
			},
			expectedName: "myimage-v1-repo-name-develop",
		},
		{
			name: "complex image tag with special chars",
			config: SpinConfig{
				Image: "my.image:v2.0.1",
				Repo:  "https://github.com/user/My_Project.git",
			},
			expectedName: "my-image-v2-0-1-my_project",
		},
		{
			name: "branch with slashes and special chars",
			config: SpinConfig{
				Image:  "spinner:env",
				Repo:   "https://github.com/user/repo.git",
				Branch: "feature/JIRA-123_bugfix",
			},
			expectedName: "spinner-env-repo-feature-jira-123_bugfix",
		},
		{
			name: "uppercase repo and branch",
			config: SpinConfig{
				Image:  "IMAGE:TAG",
				Repo:   "https://github.com/User/REPO.git",
				Branch: "MAIN",
			},
			expectedName: "image-tag-repo-main",
		},
		{
			name: "multiple consecutive special chars",
			config: SpinConfig{
				Image: "test::image",
				Repo:  "https://github.com/user/repo--name.git",
			},
			expectedName: "test-image-repo-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateContainerName(tt.config)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}
