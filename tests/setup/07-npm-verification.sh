#!/bin/bash
# Test: verify npm is installed in the container

echo "Test: npm is installed in container"

# Cleanup function
cleanup() {
  if [ -n "$TEST_CONTAINER" ]; then
    docker rm -f "$TEST_CONTAINER" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

# Start a temporary container
TEST_CONTAINER="test-npm-verify-$$"
if ! docker run -d --name "$TEST_CONTAINER" spinner:test-env tail -f /dev/null >/dev/null 2>&1; then
  echo "✗ Test failed: Could not start container"
  exit 1
fi

# Check if npm is installed
if docker exec "$TEST_CONTAINER" bash -c 'source ~/.nvm/nvm.sh && npm --version' >/dev/null 2>&1; then
  echo "✓ Test passed: npm is installed in container"
  exit 0
fi

echo "✗ Test failed: npm is not installed in container"
exit 1
