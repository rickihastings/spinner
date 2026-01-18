#!/bin/bash
# Test: setup command with --name but without --jvm-url shows error

echo "Test: Missing --jvm-url flag"

output=$(node ../../dist/cli.js setup --name test 2>&1 || true)

if echo "$output" | grep -q "Missing required flag"; then
  echo "✓ Test passed: Error message displayed for missing --jvm-url flag"
  exit 0
fi

echo "✗ Test failed: Expected error for missing --jvm-url flag"
echo "Output: $output"
exit 1
