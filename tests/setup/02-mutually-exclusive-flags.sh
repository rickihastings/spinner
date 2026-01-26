#!/bin/bash
# Test: --base-image and --dockerfile are mutually exclusive

echo "Test: Mutually exclusive flags"

output=$(../../dist/spinner setup --name test --base-image ubuntu:22.04 --dockerfile ./Dockerfile 2>&1 || true)

if echo "$output" | grep -q "mutually exclusive"; then
  echo "✓ Test passed: Error message displayed for mutually exclusive flags"
  exit 0
fi

echo "✗ Test failed: Expected error for mutually exclusive flags"
echo "Output: $output"
exit 1
