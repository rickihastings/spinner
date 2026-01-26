#!/bin/bash
# Test: container is running after spin command completes

echo "Test: Container is running"

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

# Check if container is running
if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  status=$(docker inspect -f '{{.State.Status}}' "$CONTAINER_NAME")
  if [ "$status" = "running" ]; then
    echo "✓ Test passed: Container is running"
    exit 0
  fi
fi

echo "✗ Test failed: Container is not running"
exit 1
