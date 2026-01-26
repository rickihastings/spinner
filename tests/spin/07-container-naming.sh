#!/bin/bash
# Test: container is named deterministically based on image + repo

echo "Test: Deterministic container naming (image + repo)"

# Source environment variables
source ../../.envrc

# Cleanup function
cleanup() {
  if [ -n "$CONTAINER_NAME" ]; then
    echo "Cleaning up container: $CONTAINER_NAME"
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

# Use a public test repository
# Note: assumes spinner:test-env exists from setup tests
TEST_REPO="https://github.com/octocat/Hello-World.git"
EXPECTED_NAME="spinner-test-env-hello-world"

# Run spin command
output=$(../../dist/spinner spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  exit 1
fi

# Check if container name matches expected deterministic name
if [ "$CONTAINER_NAME" = "$EXPECTED_NAME" ]; then
  echo "✓ Test passed: Container named deterministically: $CONTAINER_NAME"
  exit 0
fi

echo "✗ Test failed: Expected '$EXPECTED_NAME', got '$CONTAINER_NAME'"
exit 1
