package gcp

import (
	"strings"
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
)

func TestGenerateInstanceName(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		repo     string
		branch   string
		expected string
	}{
		{
			name:     "basic name without branch",
			image:    "my-env",
			repo:     "https://github.com/user/my-repo.git",
			branch:   "",
			expected: "spinner-my-env-my-repo",
		},
		{
			name:     "basic name with branch",
			image:    "my-env",
			repo:     "https://github.com/user/my-repo.git",
			branch:   "feature-branch",
			expected: "spinner-my-env-my-repo-feature-branch",
		},
		{
			name:     "SSH repo URL",
			image:    "default",
			repo:     "git@github.com:octocat/Hello-World.git",
			branch:   "",
			expected: "spinner-default-hello-world",
		},
		{
			name:     "uppercase characters lowercased",
			image:    "MyEnv",
			repo:     "https://github.com/user/MyRepo.git",
			branch:   "Feature-BRANCH",
			expected: "spinner-myenv-myrepo-feature-branch",
		},
		{
			name:     "special characters replaced",
			image:    "my_env.v2",
			repo:     "https://github.com/user/my_repo.git",
			branch:   "fix/bug-123",
			expected: "spinner-my-env-v2-my-repo-fix-bug-123",
		},
		{
			name:     "repo without .git suffix",
			image:    "test",
			repo:     "https://github.com/user/repo",
			branch:   "",
			expected: "spinner-test-repo",
		},
		{
			name:     "deterministic - same inputs same output",
			image:    "env1",
			repo:     "https://github.com/org/service.git",
			branch:   "main",
			expected: "spinner-env1-service-main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateInstanceName(tt.image, tt.repo, tt.branch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateInstanceNameDeterministic(t *testing.T) {
	name1 := generateInstanceName("env", "https://github.com/user/repo.git", "main")
	name2 := generateInstanceName("env", "https://github.com/user/repo.git", "main")
	assert.Equal(t, name1, name2)
}

func TestGenerateInstanceNameGCPConstraints(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		repo   string
		branch string
	}{
		{
			name:   "basic",
			image:  "env",
			repo:   "https://github.com/user/repo.git",
			branch: "",
		},
		{
			name:   "with branch",
			image:  "env",
			repo:   "https://github.com/user/repo.git",
			branch: "feature/test",
		},
		{
			name:   "long name",
			image:  "my-very-long-environment-name",
			repo:   "https://github.com/organization/very-long-repository-name.git",
			branch: "feature/very-long-branch-name-that-exceeds-limits",
		},
		{
			name:   "special characters",
			image:  "env@v2.0",
			repo:   "https://github.com/user/repo_name.git",
			branch: "fix/bug#123",
		},
	}

	gcpNameRegex := `^[a-z][a-z0-9-]*[a-z0-9]$`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateInstanceName(tt.image, tt.repo, tt.branch)

			// Must be <= 63 characters
			assert.LessOrEqual(t, len(result), maxInstanceNameLength,
				"name exceeds 63 chars: %s (len=%d)", result, len(result))

			// Must be lowercase
			assert.Equal(t, strings.ToLower(result), result,
				"name must be lowercase: %s", result)

			// Must match GCP naming pattern
			assert.Regexp(t, gcpNameRegex, result,
				"name doesn't match GCP pattern: %s", result)
		})
	}
}

func TestGenerateInstanceNameTruncation(t *testing.T) {
	// Generate a name that would exceed 63 characters
	longImage := "very-long-environment-name-for-testing"
	longRepo := "https://github.com/organization/very-long-repository-name-for-testing.git"
	longBranch := "feature/very-long-branch-name"

	result := generateInstanceName(longImage, longRepo, longBranch)

	assert.LessOrEqual(t, len(result), maxInstanceNameLength)
	assert.True(t, len(result) > 0)

	// Verify the hash suffix is present (last 8 hex chars after a hyphen)
	parts := strings.Split(result, "-")
	lastPart := parts[len(parts)-1]
	assert.Len(t, lastPart, hashSuffixLength,
		"truncated name should end with %d-char hash suffix", hashSuffixLength)

	// Verify determinism after truncation
	result2 := generateInstanceName(longImage, longRepo, longBranch)
	assert.Equal(t, result, result2)
}

func TestGenerateInstanceNameNoTruncation(t *testing.T) {
	// Short name should not be truncated or have hash suffix
	result := generateInstanceName("env", "https://github.com/u/r.git", "")
	assert.Equal(t, "spinner-env-r", result)
	assert.Less(t, len(result), maxInstanceNameLength)
}

func TestMapVMStatus(t *testing.T) {
	tests := []struct {
		name     string
		vmStatus string
		expected provider.InstanceStatus
	}{
		{
			name:     "running",
			vmStatus: "RUNNING",
			expected: provider.InstanceStatusRunning,
		},
		{
			name:     "provisioning",
			vmStatus: "PROVISIONING",
			expected: provider.InstanceStatusRunning,
		},
		{
			name:     "staging",
			vmStatus: "STAGING",
			expected: provider.InstanceStatusRunning,
		},
		{
			name:     "terminated",
			vmStatus: "TERMINATED",
			expected: provider.InstanceStatusStopped,
		},
		{
			name:     "stopped",
			vmStatus: "STOPPED",
			expected: provider.InstanceStatusStopped,
		},
		{
			name:     "suspended",
			vmStatus: "SUSPENDED",
			expected: provider.InstanceStatusStopped,
		},
		{
			name:     "stopping",
			vmStatus: "STOPPING",
			expected: provider.InstanceStatusStopped,
		},
		{
			name:     "suspending",
			vmStatus: "SUSPENDING",
			expected: provider.InstanceStatusStopped,
		},
		{
			name:     "unknown status",
			vmStatus: "UNKNOWN",
			expected: provider.InstanceStatusNone,
		},
		{
			name:     "empty status",
			vmStatus: "",
			expected: provider.InstanceStatusNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapVMStatus(tt.vmStatus)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeGCPName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase passthrough",
			input:    "spinner-env-repo",
			expected: "spinner-env-repo",
		},
		{
			name:     "uppercase to lowercase",
			input:    "Spinner-Env-Repo",
			expected: "spinner-env-repo",
		},
		{
			name:     "special chars replaced",
			input:    "spinner_env.repo@v2",
			expected: "spinner-env-repo-v2",
		},
		{
			name:     "consecutive hyphens collapsed",
			input:    "spinner---env---repo",
			expected: "spinner-env-repo",
		},
		{
			name:     "leading/trailing hyphens trimmed",
			input:    "-spinner-env-repo-",
			expected: "spinner-env-repo",
		},
		{
			name:     "starts with number gets prefix",
			input:    "123-env",
			expected: "s123-env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeGCPName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "my-image",
			expected: "my-image",
		},
		{
			name:     "uppercase converted",
			input:    "My-Image",
			expected: "my-image",
		},
		{
			name:     "special chars replaced",
			input:    "my/image@v2",
			expected: "my-image-v2",
		},
		{
			name:     "starts with number gets prefix",
			input:    "123-image",
			expected: "l123-image",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "long string truncated",
			input:    strings.Repeat("a", 100),
			expected: strings.Repeat("a", 63),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateWithHash(t *testing.T) {
	sanitized := strings.Repeat("a", 70)
	original := "original-name-for-hashing"

	result := truncateWithHash(sanitized, original)

	assert.LessOrEqual(t, len(result), maxInstanceNameLength)
	assert.True(t, strings.Contains(result, "-"))

	// Deterministic
	result2 := truncateWithHash(sanitized, original)
	assert.Equal(t, result, result2)

	// Different original produces different hash
	result3 := truncateWithHash(sanitized, "different-original")
	assert.NotEqual(t, result, result3)
}
