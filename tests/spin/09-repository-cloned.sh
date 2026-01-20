#!/bin/bash
# Test: repository is cloned into /home/spinner/workspace inside container

echo "Test: Repository cloned into /home/spinner/workspace"

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
output=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  exit 1
fi

# Wait a moment for clone to complete
sleep 2

# Check if /home/spinner/workspace exists and has content
if docker exec "$CONTAINER_NAME" test -d /home/spinner/workspace/.git; then
  echo "✓ Test passed: Repository cloned into /home/spinner/workspace"
  exit 0
fi

echo "✗ Test failed: Repository not found in /home/spinner/workspace"
exit 1
