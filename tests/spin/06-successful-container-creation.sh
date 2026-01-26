#!/bin/bash
# Test: successful container creation with valid flags

echo "Test: Successful container creation"

# Source environment variables
source ../../.envrc

# Cleanup function
cleanup() {
  echo "Cleaning up..."
  if [ -n "$CONTAINER_NAME" ]; then
    echo "Cleaning up container: $CONTAINER_NAME"
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  # Also clean up any containers matching the pattern in case extraction failed
  docker ps -a --filter "name=spinner-test-env-hello-world" --format "{{.Names}}" | xargs -r docker rm -f >/dev/null 2>&1 || true
}

trap cleanup EXIT INT TERM

# Create a temporary public test repository URL (using a real public repo)
# Note: assumes spinner:test-env exists from setup tests
TEST_REPO="https://github.com/octocat/Hello-World.git"

# Run spin command
output=$(../../dist/spinner spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Extract container name from output (macOS compatible)
CONTAINER_NAME=$(echo "$output" | grep 'Container created successfully:' | sed 's/.*Container created successfully: \([^ ]*\).*/\1/' || echo "")

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not find container name in output"
  echo "Output: $output"
  exit 1
fi

# Check if container was created
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "✓ Test passed: Container created successfully"
  exit 0
fi

echo "✗ Test failed: Container not found"
exit 1
