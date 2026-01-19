#!/bin/bash
# Test: --branch without --prompt clones repo and stays idle (no Ralph loop)

echo "Test: --branch without --prompt creates idle container"

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

# Run spin command with --branch but without --prompt
output=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" --branch test 2>&1)
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

# Check if PROMPT is not set (container should be idle)
prompt_set=$(docker inspect "$CONTAINER_NAME" -f '{{range .Config.Env}}{{println .}}{{end}}' | grep "^PROMPT=" | cut -d'=' -f1)

if [ -z "$prompt_set" ]; then
  echo "✓ Test passed: Container created without PROMPT (idle mode)"
  exit 0
else
  echo "✗ Test failed: PROMPT unexpectedly set"
  exit 1
fi
