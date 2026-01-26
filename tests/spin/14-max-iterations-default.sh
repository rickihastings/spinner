#!/bin/bash
# Test: --max-iterations defaults to 100 when not provided

echo "Test: --max-iterations defaults to 100"

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
TEST_REPO="https://github.com/octocat/Hello-World.git"

# Run spin command with --prompt but no --max-iterations
output=$(../../dist/spinner spin --image spinner:test-env --repo "$TEST_REPO" --prompt "echo test" 2>&1)
exit_code=$?

# Extract container name from output
CONTAINER_NAME=$(echo "$output" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not extract container name"
  echo "Output: $output"
  exit 1
fi

if [ $exit_code -ne 0 ]; then
  echo "✗ Test failed: Command exited with code $exit_code"
  exit 1
fi

# Check if MAX_ITERATIONS is set to 100 in the container
max_iterations=$(docker inspect "$CONTAINER_NAME" -f '{{range .Config.Env}}{{println .}}{{end}}' | grep "MAX_ITERATIONS" | cut -d'=' -f2)

if [ "$max_iterations" = "100" ]; then
  echo "✓ Test passed: MAX_ITERATIONS defaults to 100"
  exit 0
else
  echo "✗ Test failed: MAX_ITERATIONS is $max_iterations, expected 100"
  exit 1
fi
