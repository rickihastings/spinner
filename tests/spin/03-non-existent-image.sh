#!/bin/bash
# Test: non-existent Docker image exits with error

echo "Test: Non-existent Docker image"

# Run spin command with non-existent image
output=$(../../dist/spinner spin --image non-existent-image:latest --repo git@github.com:octocat/Hello-World.git 2>&1)
exit_code=$?

if echo "$output" | grep -q "Docker image 'non-existent-image:latest' not found" && [ $exit_code -eq 1 ]; then
  echo "✓ Test passed: Error message displayed and exits with code 1"
  exit 0
fi

echo "✗ Test failed: Expected error message not displayed or wrong exit code"
exit 1
