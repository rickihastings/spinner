#!/bin/bash
# Test: missing --image flag exits with error code 1

echo "Test: Missing --image flag"

# Run spin command without --image flag
output=$(../../dist/spinner spin --repo git@github.com:octocat/Hello-World.git 2>&1)
exit_code=$?

if echo "$output" | grep -q "Error: --image flag is required" && [ $exit_code -eq 1 ]; then
  echo "✓ Test passed: Error message displayed and exits with code 1"
  exit 0
fi

echo "✗ Test failed: Expected error message not displayed or wrong exit code"
exit 1
