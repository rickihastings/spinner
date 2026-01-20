#!/bin/bash
# Test: warning displayed when .npmrc missing

echo "Test: Warning displayed when .npmrc missing"

# Source environment variables
source ../../.envrc

# Cleanup function
cleanup() {
  if [ -n "$CONTAINER_NAME" ]; then
    echo "Cleaning up container: $CONTAINER_NAME"
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  # Restore .npmrc if we backed it up
  if [ -f "$HOME/.npmrc.backup" ]; then
    mv "$HOME/.npmrc.backup" "$HOME/.npmrc"
  fi
}

trap cleanup EXIT

# Backup and remove .npmrc if it exists
if [ -f "$HOME/.npmrc" ]; then
  mv "$HOME/.npmrc" "$HOME/.npmrc.backup"
fi

# Use a public test repository
# Note: assumes spinner:test-env exists from setup tests
TEST_REPO="https://github.com/octocat/Hello-World.git"

# Run spin command
output=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Check for warning message
if echo "$output" | grep -q "~/.npmrc not found, npm will use default registry"; then
  echo "✓ Test passed: Warning displayed when .npmrc missing"

  # Extract and cleanup container
  CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')
  exit 0
fi

echo "✗ Test failed: Warning not displayed"
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')
exit 1
