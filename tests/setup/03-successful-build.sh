#!/bin/bash
# Test: successful Docker image build with setup command
# This creates the spinner:test-env image that will be used by all spin tests

echo "Test: Successful Docker image build"

# Detect architecture for JVM URL
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    JVM_URL="https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.6%2B7/OpenJDK21U-jdk_x64_linux_hotspot_21.0.6_7.tar.gz"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    JVM_URL="https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.6%2B7/OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.6_7.tar.gz"
else
    echo "✗ Test failed: Unsupported architecture: $ARCH"
    exit 1
fi

# Build the test image (this will be reused by all spin tests)
output=$(node ../../dist/cli.js setup --name test-env --node-version 20 --jvm-url "$JVM_URL" 2>&1)

if echo "$output" | grep -q "Docker image built successfully"; then
  echo "✓ Test passed: Docker image built successfully"
  exit 0
fi

echo "✗ Test failed: Image build did not complete successfully"
echo "Output: $output"
exit 1
