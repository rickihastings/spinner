package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseContainerListOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ContainerListEntry
		wantErr  bool
	}{
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "  \n  ",
			expected: nil,
		},
		{
			name:  "single container",
			input: `{"ID":"abc123","Names":"my-container","Image":"spinner:test","State":"running","Labels":"spinner-managed=true"}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc123",
					Names:  []string{"my-container"},
					Image:  "spinner:test",
					State:  "running",
					Labels: map[string]string{"spinner-managed": "true"},
				},
			},
		},
		{
			name: "multiple containers",
			input: `{"ID":"abc123","Names":"container-1","Image":"spinner:test","State":"running","Labels":"spinner-managed=true"}
{"ID":"def456","Names":"container-2","Image":"spinner:test","State":"exited","Labels":"spinner-managed=true"}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc123",
					Names:  []string{"container-1"},
					Image:  "spinner:test",
					State:  "running",
					Labels: map[string]string{"spinner-managed": "true"},
				},
				{
					ID:     "def456",
					Names:  []string{"container-2"},
					Image:  "spinner:test",
					State:  "exited",
					Labels: map[string]string{"spinner-managed": "true"},
				},
			},
		},
		{
			name:  "multiple labels",
			input: `{"ID":"abc","Names":"c1","Image":"img","State":"running","Labels":"key1=val1,key2=val2"}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc",
					Names:  []string{"c1"},
					Image:  "img",
					State:  "running",
					Labels: map[string]string{"key1": "val1", "key2": "val2"},
				},
			},
		},
		{
			name:  "empty labels",
			input: `{"ID":"abc","Names":"c1","Image":"img","State":"running","Labels":""}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc",
					Names:  []string{"c1"},
					Image:  "img",
					State:  "running",
					Labels: map[string]string{},
				},
			},
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseContainerListOutput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildImage_DefaultBaseImage tests building with default base image
func TestBuildImage_DefaultBaseImage(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	config := BuildConfig{
		Name:       "test-env",
		BaseImage:  "",
		Dockerfile: "",
	}

	// Mock the BuildImage call to succeed
	mockClient.On("BuildImage", ctx, config).Return(nil)

	err := mockClient.BuildImage(ctx, config)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestBuildImage_CustomBaseImage tests building with custom base image
func TestBuildImage_CustomBaseImage(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	config := BuildConfig{
		Name:      "test-env",
		BaseImage: "node:20",
	}

	// Mock the BuildImage call to succeed
	mockClient.On("BuildImage", ctx, config).Return(nil)

	err := mockClient.BuildImage(ctx, config)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestBuildImage_WithDockerfile tests building with custom Dockerfile
func TestBuildImage_WithDockerfile(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	// Create a temporary Dockerfile
	tempDir := t.TempDir()
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")

	dockerfileContent := `FROM ubuntu:22.04
RUN apt-get update
`
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := BuildConfig{
		Name:       "test-env",
		Dockerfile: dockerfilePath,
	}

	// Mock the BuildImage call to succeed
	mockClient.On("BuildImage", ctx, config).Return(nil)

	err := mockClient.BuildImage(ctx, config)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestBuildImage_BuildFailure tests error handling when build fails
func TestBuildImage_BuildFailure(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	config := BuildConfig{
		Name: "test-env",
	}

	// Mock the BuildImage call to return an error
	expectedErr := assert.AnError
	mockClient.On("BuildImage", ctx, config).Return(expectedErr)

	err := mockClient.BuildImage(ctx, config)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockClient.AssertExpectations(t)
}

// TestBuildConfig_Variations tests various build configurations using table-driven tests
func TestBuildConfig_Variations(t *testing.T) {
	tests := []struct {
		name       string
		config     BuildConfig
		wantError  bool
		errorCheck func(*testing.T, error)
	}{
		{
			name: "valid config with name only",
			config: BuildConfig{
				Name: "test-env",
			},
			wantError: false,
		},
		{
			name: "valid config with name and base image",
			config: BuildConfig{
				Name:      "test-env",
				BaseImage: "ubuntu:22.04",
			},
			wantError: false,
		},
		{
			name: "valid config with name and dockerfile",
			config: BuildConfig{
				Name:       "test-env",
				Dockerfile: "Dockerfile.custom",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockDockerClient)
			ctx := context.Background()

			if tt.wantError {
				mockClient.On("BuildImage", ctx, tt.config).Return(assert.AnError)
			} else {
				mockClient.On("BuildImage", ctx, tt.config).Return(nil)
			}

			err := mockClient.BuildImage(ctx, tt.config)

			if tt.wantError {
				assert.Error(t, err)

				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
