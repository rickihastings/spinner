#!/bin/bash
# Test: --prompt without --branch runs Ralph loop on default branch

echo "Test: --prompt without --branch runs on default branch"

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

# Run spin command with --prompt but without --branch
output=$(node ../../dist/cli.js spin --image spinner:test-env --repo "$TEST_REPO" --prompt "echo test" 2>&1)
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

# Check if PROMPT is set but BRANCH is not set in the container
prompt_set=$(docker inspect "$CONTAINER_NAME" -f '{{range .Config.Env}}{{println .}}{{end}}' | grep "PROMPT" | cut -d'=' -f1)
branch_set=$(docker inspect "$CONTAINER_NAME" -f '{{range .Config.Env}}{{println .}}{{end}}' | grep "^BRANCH=" | cut -d'=' -f1)

if [ "$prompt_set" = "PROMPT" ] && [ -z "$branch_set" ]; then
  echo "✓ Test passed: Container created with PROMPT but no BRANCH"
  exit 0
else
  echo "✗ Test failed: PROMPT not set or BRANCH unexpectedly set"
  exit 1
fi
