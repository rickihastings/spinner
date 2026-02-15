#!/bin/bash
set -e

echo "=== Spinner Local Development Setup ==="

# Check if we're in the spinner source tree
if [ ! -f "go.mod" ] || ! grep -q "github.com/rickihastings/spinner" go.mod; then
    echo "Error: Must run from spinner project root"
    exit 1
fi

# Build the spinner binary
echo "Building spinner binary..."
mkdir -p dist
go clean -cache && go build -o dist/spinner .

# Build for linux/amd64 (for Docker/GCP)
echo "Building linux/amd64 binary..."
GOOS=linux GOARCH=amd64 go build -o dist/spinner-linux-amd64 .

# Create tarball for GCP (matches GitHub release format)
echo "Creating tarball..."
# Create temp dir, copy binary as "spinner", tar it, then remove temp dir
mkdir -p dist/tmp
cp dist/spinner-linux-amd64 dist/tmp/spinner
tar -czf dist/spinner-dev-linux-amd64.tar.gz -C dist/tmp spinner
rm -rf dist/tmp

echo ""
echo "✅ Build complete!"
echo ""
echo "Local binary will be auto-detected when running setup from source."
echo "Just run setup commands normally:"
echo ""
echo "  # Docker"
echo "  ./dist/spinner setup --name test"
echo ""
echo "  # GCP (binary will auto-upload to your state-bucket)"
echo "  ./dist/spinner setup --backend gcp --name test --state-bucket my-bucket --project my-proj --zone us-central1-a"
echo ""
