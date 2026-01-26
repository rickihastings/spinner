#!/bin/bash
# Test: container is named deterministically based on image + repo + branch

echo "Test: Deterministic container naming (image + repo + branch)"

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

# Use a public test repository with a branch
TEST_REPO="https://github.com/octocat/Hello-World.git"
TEST_BRANCH="master"
EXPECTED_NAME="spinner-test-env-hello-world-master"

# Run spin command with branch
output=$(../../dist/spinner spin --image spinner:test-env --repo "$TEST_REPO" --prompt "test" --branch "$TEST_BRANCH" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  exit 1
fi

# Check if container name matches expected deterministic name with branch
if [ "$CONTAINER_NAME" = "$EXPECTED_NAME" ]; then
  echo "✓ Test passed: Container named deterministically with branch: $CONTAINER_NAME"
  exit 0
fi

echo "✗ Test failed: Expected '$EXPECTED_NAME', got '$CONTAINER_NAME'"
exit 1
