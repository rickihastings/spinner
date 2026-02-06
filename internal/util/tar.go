package util

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
)

// TarOptions configures how the tar archive is created.
type TarOptions struct {
	// Filter determines whether a file should be included in the archive.
	// Receives the relative path and file info. Returns true to include the file.
	// If nil, all files are included.
	Filter func(relPath string, info os.FileInfo) bool
}

// CreateTar creates a tar archive from the given directory.
// The opts parameter allows customizing which files are included.
func CreateTar(contextDir string, opts *TarOptions) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	err := filepath.Walk(contextDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path for tar header
		relPath, err := filepath.Rel(contextDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Apply filter if provided
		if opts != nil && opts.Filter != nil {
			if !opts.Filter(relPath, info) {
				// Skip directories entirely if filtered out
				if info.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}
		}

		// Resolve symlink target if needed
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}

		// Use relative path as the name in the archive
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			defer func() { _ = file.Close() }()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}
