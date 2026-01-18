#!/bin/bash
# Test: container is named correctly based on repo name

echo "Test: Container naming"

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
output=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  exit 1
fi

# Check if container name starts with expected prefix (Hello-World)
if echo "$CONTAINER_NAME" | grep -q "^Hello-World-"; then
  echo "✓ Test passed: Container named correctly: $CONTAINER_NAME"
  exit 0
fi

echo "✗ Test failed: Container name does not match expected pattern: $CONTAINER_NAME"
exit 1
