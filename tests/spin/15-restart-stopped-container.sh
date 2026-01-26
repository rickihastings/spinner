#!/bin/bash
# Test: restart stopped container

echo "Test: Restart stopped container"

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
EXPECTED_NAME="spinner-test-env-hello-world"

# First run: create container
echo "Creating initial container..."
output1=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Extract container name
CONTAINER_NAME=$(echo "$output1" | sed -n 's/.*Container created successfully: \([^ ]*\).*/\1/p')

if [ -z "$CONTAINER_NAME" ]; then
  echo "✗ Test failed: Could not create initial container"
  exit 1
fi

echo "Initial container created: $CONTAINER_NAME"

# Stop the container
echo "Stopping container..."
docker stop "$CONTAINER_NAME" >/dev/null 2>&1

# Verify container is stopped
if ! docker ps -a --filter "name=$CONTAINER_NAME" --filter "status=exited" | grep -q "$CONTAINER_NAME"; then
  echo "✗ Test failed: Container is not stopped"
  exit 1
fi

echo "Container stopped"

# Second run: should restart stopped container
echo "Running spin again with same parameters..."
output2=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" 2>&1 || true)

# Check if output indicates restart
if echo "$output2" | grep -q "Container restarted: $CONTAINER_NAME"; then
  # Verify container is running again
  if docker ps --filter "name=$CONTAINER_NAME" --filter "status=running" | grep -q "$CONTAINER_NAME"; then
    echo "✓ Test passed: Container restarted successfully"
    exit 0
  else
    echo "✗ Test failed: Container not running after restart"
    exit 1
  fi
fi

echo "✗ Test failed: Container was not restarted"
echo "Output: $output2"
exit 1
