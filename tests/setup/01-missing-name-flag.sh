#!/bin/bash
# Test: setup command without --name flag shows error

echo "Test: Missing --name flag"

output=$(../../dist/spinner setup 2>&1 || true)

if echo "$output" | grep -q "Missing required flag"; then
  echo "✓ Test passed: Error message displayed for missing --name flag"
  exit 0
fi

echo "✗ Test failed: Expected error for missing --name flag"
echo "Output: $output"
exit 1
