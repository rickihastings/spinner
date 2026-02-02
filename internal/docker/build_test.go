package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
