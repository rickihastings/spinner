#!/bin/bash
# Test: spin --setup rebuilds image even if it already exists

echo "Test: --setup rebuilds existing image"

# Source environment variables
source ../../.envrc

# Cleanup function
cleanup() {
  if [ -n "$CONTAINER_NAME" ]; then
    echo "Cleaning up container: $CONTAINER_NAME"
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  # Clean up the test image
  docker rmi -f spinner:rebuild-test >/dev/null 2>&1 || true
}

trap cleanup EXIT

# Use a public test repository
TEST_REPO="https://github.com/octocat/Hello-World.git"

# First, create the image using setup command
echo "Creating initial image..."
setup_output=$(../../dist/spinner setup --name rebuild-test --base-image ubuntu:22.04 2>&1)

if ! echo "$setup_output" | grep -q "Docker image built successfully"; then
  echo "✗ Test failed: Initial image build did not complete"
  echo "Output: $setup_output"
  exit 1
fi

# Verify the image exists before running spin with --setup
if ! docker images | grep -q "spinner.*rebuild-test"; then
  echo "✗ Test failed: Initial image was not created"
  exit 1
fi

# Now run spin with --setup on the existing image
echo "Rebuilding image with --setup..."
output=$(../../dist/spinner spin --setup --image rebuild-test --base-image ubuntu:22.04 --repo "$TEST_REPO" --prompt "echo test" 2>&1)
exit_code=$?

# Check if image build was successful (this proves --setup triggered the build process)
if ! echo "$output" | grep -q "Docker image built successfully"; then
  echo "✗ Test failed: Image rebuild did not complete"
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

# Verify the image still exists
if ! docker images | grep -q "spinner.*rebuild-test"; then
  echo "✗ Test failed: Image was not present after rebuild"
  exit 1
fi

# Verify the container is running
if ! docker ps | grep -q "$CONTAINER_NAME"; then
  echo "✗ Test failed: Container is not running"
  exit 1
fi

echo "✓ Test passed: Image was rebuilt and container created successfully"
exit 0
