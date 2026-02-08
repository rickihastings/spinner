package gcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rickihastings/spinner/internal/util"
)

// uploadLocalBinary uploads the locally-built spinner binary to the state bucket
// for use during image baking. Only called when LOCAL_BUILD=true.
func uploadLocalBinary(ctx context.Context, client Client, bucket string) error {
	fmt.Println("Uploading local binary to state bucket...")

	// Find project root
	projectRoot, err := util.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Read the tarball created by dev-setup.sh
	tarballPath := filepath.Join(projectRoot, "dist", "spinner-dev-linux-amd64.tar.gz")

	data, err := os.ReadFile(tarballPath)
	if err != nil {
		return fmt.Errorf("failed to read dist/spinner-dev-linux-amd64.tar.gz\n"+
			"Make sure you ran ./scripts/dev-setup.sh first: %w", err)
	}

	// Upload to GCS
	objectPath := "local-dev/spinner-dev-linux-amd64.tar.gz"
	if err := client.WriteObject(ctx, bucket, objectPath, data); err != nil {
		return fmt.Errorf("failed to upload to gs://%s/%s: %w", bucket, objectPath, err)
	}

	fmt.Printf("✅ Uploaded local binary to gs://%s/%s\n", bucket, objectPath)

	return nil
}
