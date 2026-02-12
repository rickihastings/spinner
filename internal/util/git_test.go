package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
