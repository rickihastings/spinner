#!/bin/bash
# Test: --base-image requires --setup flag

echo "Test: --base-image requires --setup flag"

# Source environment variables
source ../../.envrc

# Use a public test repository
TEST_REPO="https://github.com/octocat/Hello-World.git"

# Run spin command with --base-image but without --setup (should fail)
output=$(../../dist/spinner spin --image test-env --base-image ubuntu:22.04 --repo "$TEST_REPO" 2>&1 || true)

if echo "$output" | grep -q "requires --setup"; then
  echo "✓ Test passed: Error message displayed for --base-image without --setup"
  exit 0
fi

# Also check for alternative error messages
if echo "$output" | grep -q "only valid with --setup"; then
  echo "✓ Test passed: Error message displayed for --base-image without --setup"
  exit 0
fi

echo "✗ Test failed: Expected error for --base-image without --setup"
echo "Output: $output"
exit 1
