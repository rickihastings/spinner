package integration

import (
	"os"
	"testing"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestMain provides global setup and teardown for all integration tests.
// This runs once before any tests start and once after all tests complete.
func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	// Global teardown: clean up any leftover spinner data directories
	// This removes ~/.spinner/spinner-test-* directories created during tests
	_ = testutil.CleanupTestSpinnerDirs()

	os.Exit(code)
}
