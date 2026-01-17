#!/bin/bash

set -e

echo "Running integration tests for docker-sandbox"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Track cleanup
TEST_IMAGE=""
TEST_CONTAINER=""

# Cleanup function
cleanup() {
    if [ -n "$TEST_CONTAINER" ]; then
        echo -e "${BLUE}Cleaning up container: $TEST_CONTAINER${NC}"
        docker rm -f "$TEST_CONTAINER" > /dev/null 2>&1 || true
    fi
    if [ -n "$TEST_IMAGE" ]; then
        echo -e "${BLUE}Cleaning up image: $TEST_IMAGE${NC}"
        docker rmi -f "$TEST_IMAGE" > /dev/null 2>&1 || true
    fi
}

# Set up trap to cleanup on exit
trap cleanup EXIT

# Helper function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"

    echo -e "${YELLOW}Running: $test_name${NC}"
    TESTS_RUN=$((TESTS_RUN + 1))

    if eval "$test_command"; then
        echo -e "${GREEN}✓ PASSED: $test_name${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ FAILED: $test_name${NC}"
        return 1
    fi
}

# Prerequisite check tests
echo ""
echo "=== Testing Prerequisite Checks ==="
echo ""

run_test "Check Docker is installed" "docker --version > /dev/null 2>&1"
run_test "Check Git is installed" "git --version > /dev/null 2>&1"
run_test "Check Claude is installed" "claude --version > /dev/null 2>&1"

# CLI basic tests
echo ""
echo "=== Testing CLI Basics ==="
echo ""

run_test "CLI --help flag" "node dist/cli.js --help | grep -q 'docker-sandbox'"
run_test "CLI --version flag" "node dist/cli.js --version | grep -q 'version'"
run_test "CLI unknown command shows error" "node dist/cli.js invalid-cmd 2>&1 | grep -q 'Unknown command'"

# Setup command validation tests
echo ""
echo "=== Testing Setup Command Validation ==="
echo ""

# Detect architecture for JVM URL
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    JVM_URL="https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.6%2B7/OpenJDK21U-jdk_x64_linux_hotspot_21.0.6_7.tar.gz"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    JVM_URL="https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.6%2B7/OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.6_7.tar.gz"
else
    echo -e "${RED}Unsupported architecture: $ARCH${NC}"
    exit 1
fi

run_test "Setup without flags shows error" "node dist/cli.js setup 2>&1 | grep -q 'Missing required flag'"
run_test "Setup with --name but no --jvm-url shows error" "node dist/cli.js setup --name test 2>&1 | grep -q 'Missing required flag'"

# Docker image build and verification tests
echo ""
echo "=== Testing Docker Image Build and Tool Installation ==="
echo ""

# Set the test image name
TEST_IMAGE="docker-sandbox:test-integration"

# Test building the Docker image with JDK from provided URL
echo ""
if run_test "Build Docker image with setup command" \
    "node dist/cli.js setup --name test-integration --node-version 20 --jvm-url \"$JVM_URL\""; then

    # Verify the image exists
    run_test "Docker image exists" "docker images -q \"$TEST_IMAGE\" | grep -q ."

    # Start a container from the image
    echo ""
    TEST_CONTAINER="test-sandbox-container-$$"
    if run_test "Start container from built image" \
        "docker run -d --name \"$TEST_CONTAINER\" \"$TEST_IMAGE\" tail -f /dev/null"; then

        # Verify tools are installed in the container
        echo ""
        run_test "Java is installed in container" \
            "docker exec \"$TEST_CONTAINER\" java --version > /dev/null 2>&1"

        run_test "Node.js is installed in container" \
            "docker exec \"$TEST_CONTAINER\" bash -c 'source ~/.nvm/nvm.sh && node --version' > /dev/null 2>&1"

        run_test "npm is installed in container" \
            "docker exec \"$TEST_CONTAINER\" bash -c 'source ~/.nvm/nvm.sh && npm --version' > /dev/null 2>&1"

        run_test "Git is installed in container" \
            "docker exec \"$TEST_CONTAINER\" git --version > /dev/null 2>&1"

        run_test "Claude is installed in container" \
            "docker exec \"$TEST_CONTAINER\" claude --version > /dev/null 2>&1"
    fi
fi

# Print results
echo ""
echo "==================================="
echo -e "Tests run: $TESTS_RUN"
echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests failed: $((TESTS_RUN - TESTS_PASSED))${NC}"
echo ""

if [ "$TESTS_PASSED" -eq "$TESTS_RUN" ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
