#!/bin/bash
# Test: spin --setup --base-image and --dockerfile are mutually exclusive

echo "Test: --base-image and --dockerfile are mutually exclusive with --setup"

# Source environment variables
source ../../.envrc

# Create a temporary custom Dockerfile
TEMP_DIR=$(mktemp -d)
CUSTOM_DOCKERFILE="$TEMP_DIR/custom.Dockerfile"

cat > "$CUSTOM_DOCKERFILE" << 'EOF'
FROM ubuntu:22.04
WORKDIR /workspace
EOF

# Cleanup function
cleanup() {
  rm -rf "$TEMP_DIR"
}

trap cleanup EXIT

# Use a public test repository
TEST_REPO="https://github.com/octocat/Hello-World.git"

# Run spin command with both --base-image and --dockerfile (should fail)
output=$(../../dist/spinner spin --setup --image test --base-image ubuntu:22.04 --dockerfile "$CUSTOM_DOCKERFILE" --repo "$TEST_REPO" 2>&1 || true)

if echo "$output" | grep -q "mutually exclusive"; then
  echo "✓ Test passed: Error message displayed for mutually exclusive flags"
  exit 0
fi

echo "✗ Test failed: Expected error for mutually exclusive flags"
echo "Output: $output"
exit 1
