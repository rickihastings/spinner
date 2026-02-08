# Testing

## Quick Start

```bash
# Run all tests
go test ./...

# Unit tests only (fast, no Docker)
go test -short ./...

# Integration tests only (requires Docker)
go test ./tests/integration/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Requirements

- **All functionality must have tests** - Every feature, bug fix, or significant change needs tests
- **Test at the right level** - Integration for end-to-end, unit for business logic
- **If it's hard to test, redesign it** - Untestable code indicates design problems

## Test Types

**Unit Tests** (`cmd/*_test.go`, `internal/*_test.go`):
- Test logic in isolation with mocked dependencies
- Fast (milliseconds), no Docker required
- Run: `go test -short ./...`

**Integration Tests** (`tests/integration/*_test.go`):
- Test CLI end-to-end with real Docker operations
- Slower, requires Docker daemon
- Run: `go test ./tests/integration/...`
- Skip in short mode automatically

## Test Utilities

The `tests/testutil` package provides reusable helpers:

**CLI Helpers** (`cli.go`):
```go
BuildCLI(t)                                    // Build binary for testing
RunCommand(t, args...)                         // Run command, capture output
RunCommandExpectSuccess(t, args...)            // Run and expect success
RunCommandExpectError(t, args...)              // Run and expect error
```

**Docker Helpers** (`docker.go`):
```go
DockerImageExists(t, image)                    // Check if image exists
DockerContainerExists(t, name)                 // Check if container exists
DockerContainerRunning(t, name)                // Check if running
RemoveDockerImage(t, image)                    // Clean up image
RemoveDockerContainer(t, name)                 // Clean up container
EnsureDockerRunning(t)                         // Verify Docker available
```

**Fixture Helpers** (`fixtures.go`):
```go
GenerateTestID()                               // Unique test identifier
GenerateTestImageTag()                         // Unique image tag
CleanupTestResources(t, image, container)      // Clean up after test
SkipIfDockerNotAvailable(t)                    // Skip if no Docker
```

## Writing Tests

**Unit test example:**
```go
func TestSetup_MissingNameFlag(t *testing.T) {
    mockClient := new(docker.MockDockerClient)
    cmd := NewSetupCommand(mockClient)

    err := cmd.Execute()

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "--name is required")
}
```

**Integration test example:**
```go
func TestSetup_SuccessfulBuild(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    testutil.SkipIfDockerNotAvailable(t)
    imageName := "spinner:test-" + testutil.GenerateTestID()
    t.Cleanup(func() {
        testutil.RemoveDockerImage(t, imageName)
    })

    output := testutil.RunCommandExpectSuccess(t, "setup", "--name", imageName)

    assert.Contains(t, output, "Successfully built")
    assert.True(t, testutil.DockerImageExists(t, imageName))
}
```

**Table-driven test example:**
```go
func TestSpin_ArgumentParsing(t *testing.T) {
    tests := []struct {
        name        string
        args        []string
        expectError bool
        errorMsg    string
    }{
        {
            name:        "missing image",
            args:        []string{"--repo", "."},
            expectError: true,
            errorMsg:    "--image is required",
        },
        {
            name:        "valid args",
            args:        []string{"--image", "test", "--repo", "."},
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewSpinCommand(mockClient)
            cmd.SetArgs(tt.args)
            err := cmd.Execute()

            if tt.expectError {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errorMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**Helper functions:**
```go
// Create helpers for repeated setup patterns
func setupTestImage(t *testing.T) (imageTag, imageName string) {
    t.Helper()
    testutil.SkipIfDockerNotAvailable(t)
    testutil.BuildCLI(t)

    imageTag = testutil.GenerateTestImageTag(t)
    imageName = "spinner:" + imageTag

    t.Cleanup(func() {
        testutil.RemoveDockerImage(t, imageName)
    })

    testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)
    return
}
```

**When to create helpers:**
- Same setup appears in 2+ tests
- Multiple steps always go together
- Need paired cleanup logic

**When not to:**
- Only used once
- Obscures what test does
- Creates leaky abstraction

**Where to put helpers:**
- Test-specific: Same file as tests
- Reusable: `tests/testutil/` package

## Best Practices

- Use `t.Helper()` in helper functions for better error reporting
- Use `t.Cleanup()` for resource cleanup (runs even if test fails)
- Write pure functions with clear inputs/outputs
- Use dependency injection for external dependencies
- Mock system calls in unit tests
