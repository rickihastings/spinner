#!/bin/bash
# Test: verify Git is installed in the container

echo "Test: Git is installed in container"

# Cleanup function
cleanup() {
  if [ -n "$TEST_CONTAINER" ]; then
    docker rm -f "$TEST_CONTAINER" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

# Start a temporary container
TEST_CONTAINER="test-git-verify-$$"
if ! docker run -d --name "$TEST_CONTAINER" spinner:test-env tail -f /dev/null >/dev/null 2>&1; then
  echo "✗ Test failed: Could not start container"
  exit 1
fi

# Check if Git is installed
if docker exec "$TEST_CONTAINER" git --version >/dev/null 2>&1; then
  echo "✓ Test passed: Git is installed in container"
  exit 0
fi

echo "✗ Test failed: Git is not installed in container"
exit 1
