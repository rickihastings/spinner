#!/bin/bash
# Test: successful Docker image build with setup command
# This creates the spinner:test-env image that will be used by all spin tests

echo "Test: Successful Docker image build"

# Build the test image with default ubuntu:22.04 base
output=$(node ../../dist/cli.js setup --name test-env 2>&1)

if echo "$output" | grep -q "Docker image built successfully"; then
  echo "✓ Test passed: Docker image built successfully"
  exit 0
fi

echo "✗ Test failed: Image build did not complete successfully"
echo "Output: $output"
exit 1
