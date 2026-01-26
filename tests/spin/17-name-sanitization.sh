#!/bin/bash
# Test: container names are properly sanitized (special chars replaced with hyphens)

echo "Test: Container name sanitization"

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

# Use SSH format repo URL with special chars that need sanitization
TEST_REPO="git@github.com:octocat/Hello-World.git"
TEST_BRANCH="feature/auth-v2"
# Expected: spinner-test-env-hello-world-feature-auth-v2
EXPECTED_NAME="spinner-test-env-hello-world-feature-auth-v2"

# Run spin command
output=$(../../dist/spinner spin --image spinner:test-env --repo "$TEST_REPO" --prompt "test" --branch "$TEST_BRANCH" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  exit 1
fi

# Check if container name is properly sanitized
if [ "$CONTAINER_NAME" = "$EXPECTED_NAME" ]; then
  echo "✓ Test passed: Container name properly sanitized: $CONTAINER_NAME"
  exit 0
fi

echo "✗ Test failed: Expected '$EXPECTED_NAME', got '$CONTAINER_NAME'"
exit 1
