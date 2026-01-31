# Spinner Test Suite

Comprehensive test suite for the Spinner CLI tool using Go's native testing framework.

## Quick Start

```bash
# Run all tests (unit + integration)
go test ./...

# Run only unit tests (fast, no Docker required)
go test -short ./...

# Run only integration tests (requires Docker)
go test ./tests/integration/...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
go test ./cmd/...
go test ./internal/docker/...
go test ./internal/prerequisites/...

# Verbose output
go test -v ./...
```

## Test Organization

### Structure

```
tests/
├── integration/          # Integration tests (end-to-end with real Docker)
│   ├── setup_test.go    # Setup command integration tests
│   └── spin_test.go     # Spin command integration tests
├── testutil/            # Test utilities and helpers
│   ├── cli.go           # CLI execution helpers
│   ├── docker.go        # Docker test helpers
│   └── fixtures.go      # Test fixtures and utilities
├── setup/               # [DEPRECATED] Bash setup tests
├── spin/                # [DEPRECATED] Bash spin tests
└── README.md            # This file

cmd/
├── setup_test.go        # Setup command unit tests
├── spin_test.go         # Spin command unit tests
└── ...

internal/
├── docker/
│   ├── build_test.go    # Docker build logic unit tests
│   ├── run_test.go      # Docker run logic unit tests
│   └── ...
└── prerequisites/
    ├── prerequisites_test.go  # Prerequisites validation unit tests
    └── ...
```

## Test Types

### Unit Tests

**Location:** Co-located with source code (`*_test.go` files)

**Purpose:**
- Test validation logic, argument parsing, and configuration handling
- Test business logic in isolation
- Fast execution (milliseconds)
- No external dependencies (Docker mocked)

**Run unit tests only:**
```bash
go test -short ./cmd/...
go test -short ./internal/...
```

**Examples:**
- `cmd/setup_test.go` - Setup command validation and argument parsing
- `cmd/spin_test.go` - Spin command validation and flag handling
- `internal/docker/build_test.go` - Build configuration logic
- `internal/docker/run_test.go` - Container naming and configuration
- `internal/prerequisites/prerequisites_test.go` - Environment validation

### Integration Tests

**Location:** `tests/integration/`

**Purpose:**
- Test end-to-end CLI behavior with real Docker operations
- Verify actual container creation, image building, and cleanup
- Validate Docker integration works correctly

**Run integration tests only:**
```bash
go test ./tests/integration/...
```

**Prerequisites:**
- Docker installed and running
- `GITHUB_TOKEN` environment variable set
- `ANTHROPIC_API_KEY` environment variable set

**Examples:**
- `setup_test.go` - Setup command end-to-end (image building, verification)
- `spin_test.go` - Spin command end-to-end (container creation, lifecycle)

## Running Tests

### All Tests

```bash
# Standard test run
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# With race detection
go test -race ./...
```

### Skip Integration Tests

Integration tests can be slow (30-60 seconds) because they interact with Docker. Skip them during development:

```bash
# Run only fast unit tests
go test -short ./...
```

Integration tests check for the `-short` flag and skip themselves automatically.

### Run Specific Tests

```bash
# Run specific test by name
go test ./cmd -run TestSetup_MissingNameFlag

# Run tests matching a pattern
go test ./tests/integration -run TestSpin_.*

# Run tests in a specific package
go test ./internal/docker/...
```

### Parallel Execution

Go tests run in parallel by default. Control parallelism:

```bash
# Run with specific parallel count
go test -p 4 ./...

# Disable parallelism (useful for debugging)
go test -p 1 ./...
```

## Coverage Analysis

**Generate coverage report:**
```bash
# Generate coverage data
go test -coverprofile=coverage.out ./...

# View coverage by function
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## Test Utilities

The `tests/testutil` package provides reusable test helpers:

### CLI Helpers (`cli.go`)

```go
// Build the CLI binary for testing
BuildCLI(t *testing.T) string

// Run a command and capture output
RunCommand(t *testing.T, args ...string) (stdout, stderr string, exitCode int)

// Run command with custom environment
RunCommandWithEnv(t *testing.T, env []string, args ...string) (stdout, stderr string, exitCode int)

// Expect successful command execution
RunCommandExpectSuccess(t *testing.T, args ...string) string

// Expect command to fail
RunCommandExpectError(t *testing.T, args ...string) string
```

### Docker Helpers (`docker.go`)

```go
// Check if Docker container exists
DockerContainerExists(t *testing.T, name string) bool

// Check if Docker container is running
DockerContainerRunning(t *testing.T, name string) bool

// Check if Docker image exists
DockerImageExists(t *testing.T, image string) bool

// Remove Docker image
RemoveDockerImage(t *testing.T, image string)

// Remove Docker container
RemoveDockerContainer(t *testing.T, name string)

// Wait for container to be running
WaitForContainer(t *testing.T, name string, timeout time.Duration) bool

// Get container ID by name
GetContainerID(t *testing.T, name string) string

// Ensure Docker is running
EnsureDockerRunning(t *testing.T)
```

### Fixture Helpers (`fixtures.go`)

```go
// Generate unique test identifier
GenerateTestID() string

// Generate unique test image tag
GenerateTestImageTag() string

// Cleanup test resources (images and containers)
CleanupTestResources(t *testing.T, image, container string)

// Skip test if Docker is not available
SkipIfDockerNotAvailable(t *testing.T)
```

## Writing Tests

### Unit Test Example

```go
func TestSetup_MissingNameFlag(t *testing.T) {
    // Arrange
    mockClient := new(docker.MockDockerClient)
    cmd := NewSetupCommand(mockClient)

    // Act
    err := cmd.Execute()

    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "--name is required")
}
```

### Integration Test Example

```go
func TestSetupIntegration_SuccessfulBuild(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup
    testutil.SkipIfDockerNotAvailable(t)
    imageName := "spinner:test-" + testutil.GenerateTestID()
    t.Cleanup(func() {
        testutil.RemoveDockerImage(t, imageName)
    })

    // Execute
    output := testutil.RunCommandExpectSuccess(t,
        "setup", "--name", imageName)

    // Verify
    assert.Contains(t, output, "Successfully built")
    assert.True(t, testutil.DockerImageExists(t, imageName))
}
```

### Table-Driven Test Example

```go
func TestSpin_ArgumentParsing(t *testing.T) {
    tests := []struct {
        name        string
        args        []string
        expectError bool
        errorMsg    string
    }{
        {
            name:        "missing image flag",
            args:        []string{"--repo", "."},
            expectError: true,
            errorMsg:    "--image is required",
        },
        {
            name:        "valid arguments",
            args:        []string{"--image", "test", "--repo", "."},
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClient := new(docker.MockDockerClient)
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

## Prerequisites for Integration Tests

Integration tests require:

1. **Docker** - Installed and running
   ```bash
   docker --version
   docker ps
   ```

2. **Environment Variables** - For Git and Claude operations
   ```bash
   export GITHUB_TOKEN="your-github-token"
   export ANTHROPIC_API_KEY="your-claude-api-key"
   ```

3. **Go Binary Built** - Tests use the compiled binary
   ```bash
   go build -o dist/spinner
   ```

Integration tests automatically skip if Docker is not available.

## Cleanup

Integration tests use `t.Cleanup()` to automatically clean up resources. If tests fail unexpectedly:

```bash
# Remove test containers
docker ps -a | grep -E "spinner-test-|test-" | awk '{print $1}' | xargs docker rm -f

# Remove test images
docker images | grep -E "spinner:test-|test-env" | awk '{print $3}' | xargs docker rmi -f
```

## CI/CD Integration

The test suite is designed for CI/CD environments:

```yaml
# Example GitHub Actions workflow
- name: Run unit tests
  run: go test -short ./...

- name: Run integration tests
  run: go test ./tests/integration/...
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

- name: Generate coverage
  run: |
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
```

## Migration from Bash Tests

**Status:** ✅ Migration complete

All 31 bash tests have been migrated to Go:
- 7 setup tests → 11 Go tests (5 integration + 6 unit)
- 24 spin tests → 54+ Go tests (20 integration + 34 unit)

**Bash tests are deprecated** and will be removed after validation. Use Go tests for all new testing needs.

### Running Legacy Bash Tests

If needed, bash tests can still be run:

```bash
# Run all bash tests
./tests/run.sh

# Run setup bash tests
./tests/setup/run-all.sh

# Run spin bash tests
./tests/spin/run-all.sh
```

**Note:** Bash tests require `npm run build` first and are maintained only for migration validation.
