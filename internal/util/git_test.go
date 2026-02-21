package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{"https://github.com/org/team/my-repo.git", "my-repo"},
		{"invalid-url", "invalid-url"},
		{"", "sandbox"},
	}

	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			result := ExtractRepoName(tt.repoURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
			result := ConvertSshToHttps(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
