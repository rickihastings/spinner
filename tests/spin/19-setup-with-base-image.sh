#!/bin/bash
# Test: spin --setup --image test --base-image ubuntu:22.04 --repo <url> builds and spins

echo "Test: Spin with --setup and --base-image builds and creates container"

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
  docker rmi -f spinner:setup-test >/dev/null 2>&1 || true
}

trap cleanup EXIT

# Use a public test repository
TEST_REPO="https://github.com/octocat/Hello-World.git"

# Run spin command with --setup and --base-image
output=$(../../dist/spinner spin --setup --image setup-test --base-image ubuntu:22.04 --repo "$TEST_REPO" --prompt "echo test" 2>&1)
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
if ! docker images | grep -q "spinner.*setup-test"; then
  echo "✗ Test failed: Image spinner:setup-test was not created"
  exit 1
fi

# Verify the container is running
if ! docker ps | grep -q "$CONTAINER_NAME"; then
  echo "✗ Test failed: Container is not running"
  exit 1
fi

echo "✓ Test passed: Image built and container created successfully"
exit 0