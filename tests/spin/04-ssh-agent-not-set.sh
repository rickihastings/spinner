#!/bin/bash
# Test: SSH_AUTH_SOCK not set exits with error

echo "Test: SSH_AUTH_SOCK not set"

# Run spin command without SSH_AUTH_SOCK
# Note: assumes spinner:test-env exists from setup tests
output=$(SSH_AUTH_SOCK="" node ../../dist/cli.js spin --image spinner:test-env --repo git@github.com:octocat/Hello-World.git 2>&1)
exit_code=$?

if echo "$output" | grep -q "SSH agent not running" && [ $exit_code -eq 1 ]; then
  echo "✓ Test passed: Error message displayed and exits with code 1"
  exit 0
fi

echo "✗ Test failed: Expected error message not displayed or wrong exit code"
exit 1
