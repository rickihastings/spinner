#!/bin/bash
# Test: verify the spinner:test-env image exists

echo "Test: Docker image exists"

if docker images -q "spinner:test-env" | grep -q "."; then
  echo "✓ Test passed: Docker image spinner:test-env exists"
  exit 0
fi

echo "✗ Test failed: Docker image spinner:test-env not found"
exit 1
