#!/bin/bash
# Master test runner for Spinner CLI
# Runs all integration tests: setup and spin commands

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Overall test counters
TOTAL_SUITES=0
PASSED_SUITES=0
FAILED_SUITES=0

echo ""
echo "=========================================="
echo "  Spinner Integration Test Suite"
echo "=========================================="
echo ""

# Function to run a test suite
run_suite() {
    local suite_name="$1"
    local suite_script="$2"

    TOTAL_SUITES=$((TOTAL_SUITES + 1))

    echo ""
    echo -e "${CYAN}=== Running $suite_name ===${NC}"
    echo ""

    if bash "$suite_script"; then
        PASSED_SUITES=$((PASSED_SUITES + 1))
        echo ""
        echo -e "${GREEN}✓ $suite_name: PASSED${NC}"
        return 0
    else
        FAILED_SUITES=$((FAILED_SUITES + 1))
        echo ""
        echo -e "${RED}✗ $suite_name: FAILED${NC}"
        echo ""
        echo "=========================================="
        echo "  Test Suite Summary"
        echo "=========================================="
        echo -e "Total suites: $TOTAL_SUITES"
        echo -e "${GREEN}Passed: $PASSED_SUITES${NC}"
        echo -e "${RED}Failed: $FAILED_SUITES${NC}"
        echo ""
        echo -e "${RED}Exiting due to test suite failure.${NC}"
        exit 1
    fi
}

# Run setup command tests
# This creates the spinner:test-env image that spin tests will use
run_suite "Setup Command Tests" "./setup/run-all.sh"

# Run spin command tests
run_suite "Spin Command Tests" "./spin/run-all.sh"

# Print overall results
echo ""
echo "=========================================="
echo "  Test Suite Summary"
echo "=========================================="
echo -e "Total suites: $TOTAL_SUITES"
echo -e "${GREEN}Passed: $PASSED_SUITES${NC}"
echo -e "${RED}Failed: $FAILED_SUITES${NC}"
echo ""

if [ $FAILED_SUITES -eq 0 ]; then
    echo -e "${GREEN}✓ All test suites passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some test suites failed.${NC}"
    exit 1
fi
