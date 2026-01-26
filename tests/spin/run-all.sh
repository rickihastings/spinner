#!/bin/bash
# Master test runner script for spin command

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Make all test scripts executable
chmod +x *.sh

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counter for test results
PASSED=0
TOTAL=0

# Test timeout in seconds
TEST_TIMEOUT=30

# Function to run a test with timeout
run_with_timeout() {
  local test_script="$1"
  local timeout="$2"

  # Run the test in background
  bash "$test_script" &
  local test_pid=$!

  # Wait for the test with timeout
  local elapsed=0
  while kill -0 $test_pid 2>/dev/null; do
    if [ $elapsed -ge $timeout ]; then
      echo -e "${RED}✗ Test timed out after ${timeout}s, killing process${NC}"
      kill -9 $test_pid 2>/dev/null || true
      wait $test_pid 2>/dev/null || true
      return 124  # timeout exit code
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done

  # Get the test exit code
  wait $test_pid
  return $?
}

echo "======================================"
echo "Running Spin Command Integration Tests"
echo "======================================"
echo ""

# Run each test script - exit immediately on first failure
for test in $(ls [0-9]*.sh | sort); do
  TOTAL=$((TOTAL + 1))
  echo "Running $test..."

  if ! run_with_timeout "$test" $TEST_TIMEOUT; then
    echo -e "${RED}✗ $test failed${NC}"
    # Clean up any containers created by the failed test
    docker ps -a --filter "name=hello-world" --format "{{.Names}}" | xargs -r docker rm -f >/dev/null 2>&1 || true
    docker ps -a --filter "name=spinner-" --format "{{.Names}}" | xargs -r docker rm -f >/dev/null 2>&1 || true
    echo ""
    echo "======================================"
    echo "Test Results"
    echo "======================================"
    echo -e "Total: $TOTAL"
    echo -e "${GREEN}Passed: $PASSED${NC}"
    echo -e "${RED}Failed at: $test${NC}"
    echo ""
    echo -e "${RED}Exiting test suite due to failure.${NC}"
    # Cleanup before exiting
    echo "Cleaning up any leftover test containers..."
    docker ps -a --filter "name=hello-world" --format "{{.Names}}" | while read container; do
      if [ -n "$container" ]; then
        docker rm -f "$container" >/dev/null 2>&1 || true
      fi
    done
    docker ps -a --filter "name=spinner-" --format "{{.Names}}" | while read container; do
      if [ -n "$container" ]; then
        docker rm -f "$container" >/dev/null 2>&1 || true
      fi
    done
    docker ps -a --filter "name=test-" --format "{{.Names}}" | while read container; do
      if [ -n "$container" ]; then
        docker rm -f "$container" >/dev/null 2>&1 || true
      fi
    done
    exit 1
  fi

  PASSED=$((PASSED + 1))
  echo -e "${GREEN}✓ $test passed${NC}"
  echo ""
done

# Cleanup any leftover containers
echo "Cleaning up any leftover test containers..."
# Stop and remove Hello-World containers
docker ps -a --filter "name=hello-world" --format "{{.Names}}" | while read container; do
  if [ -n "$container" ]; then
    echo "Stopping and removing: $container"
    docker stop "$container" >/dev/null 2>&1 || true
    docker rm -f "$container" >/dev/null 2>&1 || true
  fi
done
# Stop and remove spinner containers
docker ps -a --filter "name=spinner-" --format "{{.Names}}" | while read container; do
  if [ -n "$container" ]; then
    echo "Stopping and removing: $container"
    docker stop "$container" >/dev/null 2>&1 || true
    docker rm -f "$container" >/dev/null 2>&1 || true
  fi
done
# Stop and remove test containers
docker ps -a --filter "name=test-" --format "{{.Names}}" | while read container; do
  if [ -n "$container" ]; then
    echo "Stopping and removing: $container"
    docker stop "$container" >/dev/null 2>&1 || true
    docker rm -f "$container" >/dev/null 2>&1 || true
  fi
done

# Cleanup any leftover images
echo "Cleaning up any leftover images..."
# Remove test images (those created during setup tests)
docker images --filter "reference=test-*" --format "{{.Repository}}:{{.Tag}}" | while read image; do
  if [ -n "$image" ]; then
    echo "Removing image: $image"
    docker rmi -f "$image" >/dev/null 2>&1 || true
  fi
done
# Remove any dangling images created during tests
docker image prune -f >/dev/null 2>&1 || true

echo "======================================"
echo "Test Results"
echo "======================================"
echo -e "Total: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: 0${NC}"
echo ""
echo -e "${GREEN}All tests passed!${NC}"
exit 0
