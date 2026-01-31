# Testing Guidelines

## Test Coverage Requirements

- **All functionality must have tests**: Every new feature, bug fix, or significant code change must include appropriate tests
- **Test at the right level**: Use integration tests for end-to-end scenarios, unit tests for business logic
- **No untested code**: If code cannot be easily tested, it's a sign the design needs improvement

## Test Organization

### Test Types

**Unit Tests** (`cmd/*_test.go`, `internal/*_test.go`):
- Test individual functions and logic in isolation
- Use mocks for external dependencies (Docker, file system, etc.)
- Fast execution (no Docker or external services)
- Run with: `go test ./cmd/... ./internal/...`

**Integration Tests** (`tests/integration/*_test.go`):
- Test end-to-end CLI behavior with real Docker operations
- Verify actual container creation, image building, and cleanup
- Slower execution (requires Docker daemon)
- Run with: `go test ./tests/integration/...`
- Skip with short flag: `go test -short ./...`

### Test Structure

```
tests/
├── testutil/           # Reusable test utilities
│   ├── docker.go      # Docker helpers (low-level)
│   ├── cli.go         # CLI execution helpers
│   └── fixtures.go    # Test data generators
└── integration/       # Integration tests
    ├── setup_test.go  # Setup command integration tests
    └── spin_test.go   # Spin command integration tests
```

## Writing Tests

### Use Helper Functions to Avoid Boilerplate

Create helper functions for repeated setup patterns:

```go
// Helper extracts common setup logic
func setupTestImage(t *testing.T, setupArgs ...string) (imageTag string, imageName string) {
    t.Helper()
    testutil.SkipIfDockerNotAvailable(t)
    testutil.BuildCLI(t)

    imageTag = testutil.GenerateTestImageTag(t)
    imageName = "spinner:" + imageTag

    t.Cleanup(func() {
        testutil.RemoveDockerImage(t, imageName)
    })

    args := append([]string{"setup", "--name", imageTag}, setupArgs...)
    testutil.RunCommandExpectSuccess(t, args...)

    return imageTag, imageName
}

// Tests become concise and focused
func TestSetup_InstalledTools(t *testing.T) {
    tests := []struct {
        name            string
        command         []string
        wantOutputMatch string
    }{
        {name: "git installed", command: []string{"git", "--version"}, wantOutputMatch: "git version"},
        {name: "claude installed", command: []string{"claude", "--version"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, imageName := setupTestImage(t)
            containerName := runContainerWithImage(t, imageName)
            output := execInContainer(t, containerName, tt.command...)

            if tt.wantOutputMatch != "" {
                assert.Contains(t, output, tt.wantOutputMatch)
            }
        })
    }
}
```

### Use Table-Driven Tests

For testing similar scenarios with different inputs, use table-driven tests with structs defining test cases.

### When to Create Helper Functions

**Create a helper when:**
- The same setup pattern appears in 2+ tests
- Setup involves multiple steps that always go together
- Cleanup logic needs to be paired with setup

**Don't create a helper when:**
- It's only used once
- It obscures what the test is actually doing
- The abstraction is leaky or confusing

**Where to put helpers:**
- Test-specific helpers: Same file as the tests (e.g., `setup_test.go`)
- Reusable across multiple test files: `tests/testutil/` package

## Running Tests

```bash
# All tests
go test ./...

# Unit tests only (fast)
go test -short ./...

# Integration tests only
go test ./tests/integration/...

# Specific test file
go test ./tests/integration/setup_test.go -v

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Verbose output
go test -v ./...
```

## Testability Guidelines

- Write pure functions with clear inputs/outputs
- Avoid side effects where possible
- Use dependency injection for external dependencies
- Mock system calls in unit tests
- Use `t.Helper()` in helper functions for better error reporting
- Use `t.Cleanup()` for resource cleanup (runs even if test fails)
