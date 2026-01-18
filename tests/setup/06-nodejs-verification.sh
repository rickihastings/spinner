#!/bin/bash
# Test: verify Node.js is installed in the container

echo "Test: Node.js is installed in container"

# Cleanup function
cleanup() {
  if [ -n "$TEST_CONTAINER" ]; then
    docker rm -f "$TEST_CONTAINER" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

# Start a temporary container
TEST_CONTAINER="test-node-verify-$$"
if ! docker run -d --name "$TEST_CONTAINER" spinner:test-env tail -f /dev/null >/dev/null 2>&1; then
  echo "✗ Test failed: Could not start container"
  exit 1
fi

# Check if Node.js is installed
if docker exec "$TEST_CONTAINER" bash -c 'source ~/.nvm/nvm.sh && node --version' >/dev/null 2>&1; then
  echo "✓ Test passed: Node.js is installed in container"
  exit 0
fi

echo "✗ Test failed: Node.js is not installed in container"
exit 1
