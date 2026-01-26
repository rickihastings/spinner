#!/bin/bash
# Test: private repository clone with GITHUB_TOKEN

echo "Test: Private repository clone"

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

# Use this private repository
# Note: assumes spinner:test-env exists from setup tests
TEST_REPO="https://github.com/rickihastings/spinner.git"

# Run spin command
output=$(../../dist/spinner spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  echo "Output: $output"
  exit 1
fi

# Wait a moment for clone to complete
sleep 3

# Check if /home/spinner/workspace exists and has content
if docker exec "$CONTAINER_NAME" test -d /home/spinner/workspace/.git; then
  echo "✓ Test passed: Private repository cloned successfully"
  exit 0
fi

echo "✗ Test failed: Private repository not cloned"
exit 1
