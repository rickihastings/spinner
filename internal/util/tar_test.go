package util

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateTar_Success tests creating a tar archive from a directory
func TestCreateTar_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create some test files
	dockerfile := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(dockerfile, []byte("FROM ubuntu:22.04"), 0644); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tempDir, "scripts")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(subDir, "startup.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create tar archive
	tarReader, err := CreateTar(tempDir, nil)
	assert.NoError(t, err)
	assert.NotNil(t, tarReader)

	// Verify tar contents
	tr := tar.NewReader(tarReader)
	files := make(map[string]bool)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		assert.NoError(t, err)

		files[header.Name] = true
	}

	assert.True(t, files["Dockerfile"], "Dockerfile should be in tar")
	assert.True(t, files["scripts"], "scripts directory should be in tar")
	assert.True(t, files["scripts/startup.sh"], "startup.sh should be in tar")
}

// TestCreateTar_EmptyDir tests creating tar from empty directory
func TestCreateTar_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	tarReader, err := CreateTar(tempDir, nil)
	assert.NoError(t, err)
	assert.NotNil(t, tarReader)

	// Verify tar is empty (no entries)
	tr := tar.NewReader(tarReader)
	_, err = tr.Next()
	assert.Equal(t, io.EOF, err)
}

// TestCreateTar_NonExistentDir tests error handling for non-existent directory
func TestCreateTar_NonExistentDir(t *testing.T) {
	_, err := CreateTar("/nonexistent/path", nil)
	assert.Error(t, err)
}

// TestCreateTar_WithFilter tests filtering files during tar creation
func TestCreateTar_WithFilter(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tempDir, "include.txt"), []byte("included"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "exclude.log"), []byte("excluded"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "also-include.txt"), []byte("also included"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create tar with filter that excludes .log files
	opts := &TarOptions{
		Filter: func(relPath string, info os.FileInfo) bool {
			return !strings.HasSuffix(relPath, ".log")
		},
	}

	tarReader, err := CreateTar(tempDir, opts)
	assert.NoError(t, err)

	// Verify tar contents
	tr := tar.NewReader(tarReader)
	files := make(map[string]bool)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		assert.NoError(t, err)

		files[header.Name] = true
	}

	assert.True(t, files["include.txt"], "include.txt should be in tar")
	assert.True(t, files["also-include.txt"], "also-include.txt should be in tar")
	assert.False(t, files["exclude.log"], "exclude.log should NOT be in tar")
}

// TestCreateTar_FilterSkipsDirectory tests that filtering a directory skips its contents
func TestCreateTar_FilterSkipsDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory structure
	includeDir := filepath.Join(tempDir, "include")
	excludeDir := filepath.Join(tempDir, "exclude")

	if err := os.MkdirAll(includeDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(excludeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Add files to both directories
	if err := os.WriteFile(filepath.Join(includeDir, "file.txt"), []byte("included"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(excludeDir, "file.txt"), []byte("excluded"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create tar with filter that excludes 'exclude' directory
	opts := &TarOptions{
		Filter: func(relPath string, info os.FileInfo) bool {
			return !strings.HasPrefix(relPath, "exclude")
		},
	}

	tarReader, err := CreateTar(tempDir, opts)
	assert.NoError(t, err)

	// Verify tar contents
	tr := tar.NewReader(tarReader)
	files := make(map[string]bool)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		assert.NoError(t, err)

		files[header.Name] = true
	}

	assert.True(t, files["include"], "include directory should be in tar")
	assert.True(t, files["include/file.txt"], "include/file.txt should be in tar")
	assert.False(t, files["exclude"], "exclude directory should NOT be in tar")
	assert.False(t, files["exclude/file.txt"], "exclude/file.txt should NOT be in tar")
}

// TestCreateTar_FilterBySize tests filtering files by size
func TestCreateTar_FilterBySize(t *testing.T) {
	tempDir := t.TempDir()

	// Create files of different sizes
	smallFile := filepath.Join(tempDir, "small.txt")
	largeFile := filepath.Join(tempDir, "large.txt")

	if err := os.WriteFile(smallFile, []byte("small"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(largeFile, make([]byte, 1000), 0644); err != nil {
		t.Fatal(err)
	}

	// Create tar with filter that excludes files larger than 100 bytes
	opts := &TarOptions{
		Filter: func(relPath string, info os.FileInfo) bool {
			return info.IsDir() || info.Size() <= 100
		},
	}

	tarReader, err := CreateTar(tempDir, opts)
	assert.NoError(t, err)

	// Verify tar contents
	tr := tar.NewReader(tarReader)
	files := make(map[string]bool)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		assert.NoError(t, err)

		files[header.Name] = true
	}

	assert.True(t, files["small.txt"], "small.txt should be in tar")
	assert.False(t, files["large.txt"], "large.txt should NOT be in tar")
}

// TestCreateTar_Symlinks tests that symlinks are preserved with correct targets
func TestCreateTar_Symlinks(t *testing.T) {
	tempDir := t.TempDir()

	// Create a target file
	targetDir := filepath.Join(tempDir, "pkg")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(targetDir, "bin.js"), []byte("#!/usr/bin/env node"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink (mimics node_modules/.bin/husky -> ../husky/bin.js)
	binDir := filepath.Join(tempDir, ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink("../pkg/bin.js", filepath.Join(binDir, "tool")); err != nil {
		t.Fatal(err)
	}

	tarReader, err := CreateTar(tempDir, nil)
	assert.NoError(t, err)

	// Verify tar contents
	tr := tar.NewReader(tarReader)
	found := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		assert.NoError(t, err)

		if header.Name == ".bin/tool" {
			found = true

			assert.Equal(t, tar.TypeSymlink, int32(header.Typeflag), "should be a symlink entry")
			assert.Equal(t, "../pkg/bin.js", header.Linkname, "symlink target should be preserved")
		}
	}

	assert.True(t, found, ".bin/tool symlink should be in tar")
}
