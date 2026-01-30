package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateDockerfile_DefaultBaseImage tests Dockerfile generation with default base image
func TestGenerateDockerfile_DefaultBaseImage(t *testing.T) {
	config := DockerfileConfig{
		BaseImage: "ubuntu:22.04",
	}

	content, err := GenerateDockerfile(config)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "FROM ubuntu:22.04")
}

// TestGenerateDockerfile_CustomBaseImage tests Dockerfile generation with custom base image
func TestGenerateDockerfile_CustomBaseImage(t *testing.T) {
	config := DockerfileConfig{
		BaseImage: "node:20",
	}

	content, err := GenerateDockerfile(config)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "FROM node:20")
}

// TestGenerateDockerfile_TemplateNotFound tests error handling when template doesn't exist
func TestGenerateDockerfile_TemplateNotFound(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.Chdir(originalWd) }()

	// Change to a temporary directory without templates
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	config := DockerfileConfig{
		BaseImage: "ubuntu:22.04",
	}

	_, err = GenerateDockerfile(config)

	assert.Error(t, err)
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

// TestCopyFile_Success tests successful file copying
func TestCopyFile_Success(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	err := copyFile(srcPath, dstPath)

	assert.NoError(t, err)

	// Verify content was copied
	copiedContent, err := os.ReadFile(dstPath)
	assert.NoError(t, err)
	assert.Equal(t, content, copiedContent)
}

// TestCopyFile_SourceNotFound tests error handling when source doesn't exist
func TestCopyFile_SourceNotFound(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	err := copyFile(srcPath, dstPath)

	assert.Error(t, err)
}

// TestCopyFile_InvalidDestination tests error handling when destination is invalid
func TestCopyFile_InvalidDestination(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")

	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Use a path that can't be written (e.g., non-existent parent directory)
	dstPath := filepath.Join(tempDir, "nonexistent-dir", "dest.txt")

	err := copyFile(srcPath, dstPath)

	assert.Error(t, err)
}

// TestResolveTemplatePath_RelativeToCurrentDir tests path resolution from current directory
func TestResolveTemplatePath_RelativeToCurrentDir(t *testing.T) {
	// Create a temporary file in the current directory structure
	tempDir := t.TempDir()

	testPath := filepath.Join(tempDir, "test-template.txt")
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.Chdir(originalWd) }()

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	resolvedPath, err := resolveTemplatePath("test-template.txt")

	assert.NoError(t, err)
	assert.Equal(t, "test-template.txt", resolvedPath)
}

// TestFindProjectRoot_Success tests finding project root with go.mod
func TestFindProjectRoot_Success(t *testing.T) {
	// This test assumes we're running from within the project
	// which should have a go.mod file at the root
	projectRoot, err := findProjectRoot()

	assert.NoError(t, err)
	assert.NotEmpty(t, projectRoot)

	// Verify go.mod exists at the root
	goModPath := filepath.Join(projectRoot, "go.mod")
	_, err = os.Stat(goModPath)
	assert.NoError(t, err)
}

// TestFindProjectRoot_NotFound tests error when project root can't be found
func TestFindProjectRoot_NotFound(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.Chdir(originalWd) }()

	// Change to root directory where go.mod won't exist
	if err := os.Chdir("/"); err != nil {
		t.Skip("Cannot change to root directory")
	}

	_, err = findProjectRoot()

	assert.Error(t, err)
	assert.Equal(t, os.ErrNotExist, err)
}
