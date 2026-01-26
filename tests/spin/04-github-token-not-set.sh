#!/bin/bash
# Test: GITHUB_TOKEN not set exits with error

echo "Test: GITHUB_TOKEN not set"

# Run spin command without GITHUB_TOKEN
# Note: assumes spinner:test-env exists from setup tests
output=$(GITHUB_TOKEN="" ../../dist/spinner spin --image spinner:test-env --repo git@github.com:octocat/Hello-World.git 2>&1)
exit_code=$?

if echo "$output" | grep -q "GITHUB_TOKEN environment variable not set" && [ $exit_code -eq 1 ]; then
  echo "✓ Test passed: Error message displayed and exits with code 1"
  exit 0
fi

echo "✗ Test failed: Expected error message not displayed or wrong exit code"
exit 1
