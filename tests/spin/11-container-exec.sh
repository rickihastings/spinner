#!/bin/bash
# Test: container can be exec'd into with bash

echo "Test: Container can be exec'd into with bash"

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

# Run spin command
output=$(../../dist/spinner spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  exit 1
fi

# Try to exec into container
if docker exec "$CONTAINER_NAME" bash -c "pwd" >/dev/null 2>&1; then
  echo "✓ Test passed: Can exec into container with bash"
  exit 0
fi

echo "✗ Test failed: Cannot exec into container"
exit 1
