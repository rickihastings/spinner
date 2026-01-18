#!/bin/bash
# Test: missing --repo flag exits with error code 1

echo "Test: Missing --repo flag"

# Run spin command without --repo flag
output=$(node ../../dist/cli.js spin --image spinner:test 2>&1)
exit_code=$?

if echo "$output" | grep -q "Error: --repo flag is required" && [ $exit_code -eq 1 ]; then
  echo "✓ Test passed: Error message displayed and exits with code 1"
  exit 0
fi

echo "✗ Test failed: Expected error message not displayed or wrong exit code"
exit 1
