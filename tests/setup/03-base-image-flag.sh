#!/bin/bash
# Test: successful Docker image build with --base-image flag

echo "Test: Build with --base-image flag"

output=$(../../dist/spinner setup --name test-base-image --base-image ubuntu:22.04 2>&1)

if echo "$output" | grep -q "Docker image built successfully"; then
  echo "✓ Test passed: Docker image built successfully with --base-image"
  # Cleanup
  docker rmi spinner:test-base-image >/dev/null 2>&1 || true
  exit 0
fi

echo "✗ Test failed: Image build with --base-image did not complete successfully"
echo "Output: $output"
exit 1
