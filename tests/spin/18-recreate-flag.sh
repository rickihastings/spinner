#!/bin/bash
# Test: --recreate flag removes and recreates container

echo "Test: --recreate flag removes and recreates container"

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

# Verify container is running
if ! docker ps --filter "name=$CONTAINER_NAME" --filter "status=running" | grep -q "$CONTAINER_NAME"; then
  echo "✗ Test failed: Container is not running"
  exit 1
fi

# Get container ID before recreate
CONTAINER_ID_BEFORE=$(docker inspect -f '{{.Id}}' "$CONTAINER_NAME")
echo "Initial container created: $CONTAINER_NAME (ID: ${CONTAINER_ID_BEFORE:0:12})"

# Second run: use --recreate flag to force recreation
echo "Running spin with --recreate flag..."
output2=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" --recreate 2>&1 || true)

# Get container ID after recreate
CONTAINER_ID_AFTER=$(docker inspect -f '{{.Id}}' "$CONTAINER_NAME" 2>/dev/null || echo "")

if [ -z "$CONTAINER_ID_AFTER" ]; then
  echo "✗ Test failed: Container does not exist after recreate"
  exit 1
fi

# Check if container was recreated (different container ID)
if [ "$CONTAINER_ID_BEFORE" = "$CONTAINER_ID_AFTER" ]; then
  echo "✗ Test failed: Container was not recreated (same ID)"
  exit 1
fi

# Check if output indicates creation (not reuse)
if echo "$output2" | grep -q "Container created successfully: $CONTAINER_NAME"; then
  echo "✓ Test passed: Container was recreated with new ID (${CONTAINER_ID_AFTER:0:12})"
  exit 0
fi

echo "✗ Test failed: Output did not indicate container creation"
echo "Output: $output2"
exit 1
