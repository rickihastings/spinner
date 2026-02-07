package docker

import (
	"io"
	"os"
	"path/filepath"

	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/rickihastings/spinner/internal/util"
)

// BuildConfig contains configuration for building a Docker image.
type BuildConfig struct {
	Name       string
	BaseImage  string
	Dockerfile string
}

// buildFile defines a file to copy into the Docker build context
type buildFile struct {
	// Source path relative to the templates directory
	src string
	// Destination path relative to the build context templates directory
	dst string
}

// buildFiles lists all files to copy into the Docker build context
var buildFiles = []buildFile{
	{src: "scripts/startup.sh", dst: "scripts/startup.sh"},
	{src: "scripts/install_spinner.sh", dst: "scripts/install_spinner.sh"},
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

// createBuildContextTar creates a tar archive from the given directory.
// The archive is suitable for use with Docker's ImageBuild API.
// It respects .dockerignore patterns in the context directory.
func createBuildContextTar(contextDir string) (io.Reader, error) {
	filter, err := dockerIgnoreFilter(contextDir)
	if err != nil {
		return nil, err
	}

	return util.CreateTar(contextDir, &util.TarOptions{Filter: filter})
}

// dockerIgnoreFilter reads .dockerignore from contextDir and returns a filter
// function that excludes matching paths. If no .dockerignore exists, it returns
// a filter that includes all files.
func dockerIgnoreFilter(contextDir string) (func(string, os.FileInfo) bool, error) {
	ignorePath := filepath.Join(contextDir, ".dockerignore")

	f, err := os.Open(ignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No .dockerignore, include everything
			return nil, nil
		}

		return nil, err
	}

	defer func() { _ = f.Close() }()

	patterns, err := ignorefile.ReadAll(f)
	if err != nil {
		return nil, err
	}

	pm, err := patternmatcher.New(patterns)
	if err != nil {
		return nil, err
	}

	return func(relPath string, info os.FileInfo) bool {
		// Use filepath separator-normalized path for matching
		matched, _ := pm.MatchesOrParentMatches(relPath)
		return !matched
	}, nil
}
