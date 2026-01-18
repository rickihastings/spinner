#!/bin/bash
# Run all setup tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Cleaning up existing test image..."
# Remove the test-env image to ensure a fresh rebuild
docker rmi -f spinner:test-env >/dev/null 2>&1 || true
echo ""

echo "Running all setup tests..."
echo ""

PASSED_TESTS=0
TOTAL_TESTS=0

# Run each test in order - exit immediately on first failure
for test_file in $(ls -1 [0-9]*.sh | sort); do
  TOTAL_TESTS=$((TOTAL_TESTS + 1))

  echo "Running $(basename "$test_file")..."

  # Exit immediately if test fails
  if ! bash "$test_file"; then
    echo ""
    echo "======================================"
    echo "Setup Tests Summary"
    echo "======================================"
    echo "Total: $TOTAL_TESTS"
    echo "Passed: $PASSED_TESTS"
    echo "Failed at: $(basename "$test_file")"
    echo ""
    echo "Exiting test suite due to failure."
    exit 1
  fi

  PASSED_TESTS=$((PASSED_TESTS + 1))
  echo ""
done

# Print summary
echo "======================================"
echo "Setup Tests Summary"
echo "======================================"
echo "Total: $TOTAL_TESTS"
echo "Passed: $PASSED_TESTS"
echo "Failed: 0"
echo ""

echo "All setup tests passed!"
exit 0
