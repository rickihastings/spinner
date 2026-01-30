#!/bin/bash
# Test: spin --setup --image test --dockerfile ./custom.Dockerfile --repo <url> uses custom Dockerfile

echo "Test: Spin with --setup and --dockerfile uses custom Dockerfile"

# Source environment variables
source ../../.envrc

# Create a temporary custom Dockerfile
TEMP_DIR=$(mktemp -d)
CUSTOM_DOCKERFILE="$TEMP_DIR/custom.Dockerfile"

cat > "$CUSTOM_DOCKERFILE" << 'EOF'
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
EOF

# Cleanup function
cleanup() {
  if [ -n "$CONTAINER_NAME" ]; then
    echo "Cleaning up container: $CONTAINER_NAME"
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  # Clean up the test image
  docker rmi -f spinner:dockerfile-test >/dev/null 2>&1 || true
  # Clean up temporary Dockerfile
  rm -rf "$TEMP_DIR"
}

trap cleanup EXIT

# Use a public test repository
TEST_REPO="https://github.com/octocat/Hello-World.git"

# Run spin command with --setup and --dockerfile
output=$(../../dist/spinner spin --setup --image dockerfile-test --dockerfile "$CUSTOM_DOCKERFILE" --repo "$TEST_REPO" --prompt "echo test" 2>&1)
exit_code=$?

# Check if image build was successful
if ! echo "$output" | grep -q "Docker image built successfully"; then
  echo "✗ Test failed: Image build did not complete"
  echo "Output: $output"
  exit 1
fi

# Extract container name from output
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  echo "Output: $output"
  exit 1
fi

if [ $exit_code -ne 0 ]; then
  echo "✗ Test failed: Command exited with code $exit_code"
  exit 1
fi

# Verify the image was created
if ! docker images | grep -q "spinner.*dockerfile-test"; then
  echo "✗ Test failed: Image spinner:dockerfile-test was not created"
  exit 1
fi

# Verify the container is running
if ! docker ps | grep -q "$CONTAINER_NAME"; then
  echo "✗ Test failed: Container is not running"
  exit 1
fi

echo "✓ Test passed: Custom Dockerfile used and container created successfully"
exit 0