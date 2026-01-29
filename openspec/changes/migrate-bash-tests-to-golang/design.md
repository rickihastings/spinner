# Design: Golang Test Migration

## Architecture Overview

This change introduces a hybrid testing strategy with three distinct test layers:

```
┌─────────────────────────────────────────────────────┐
│                   CLI Commands                       │
│              (cmd/setup.go, cmd/spin.go)            │
└─────────────────────────────────────────────────────┘
                        │
                        ├─► Unit Tests (cmd/*_test.go)
                        │   • Argument validation
                        │   • Flag parsing
                        │   • Error handling
                        │   • Mocked dependencies
                        │
┌─────────────────────────────────────────────────────┐
│              Business Logic Layer                    │
│      (internal/docker/*, internal/prerequisites/*)  │
└─────────────────────────────────────────────────────┘
                        │
                        ├─► Unit Tests (internal/*_test.go)
                        │   • Docker client mocking
                        │   • Validation logic
                        │   • Pure function testing
                        │
┌─────────────────────────────────────────────────────┐
│                Integration Layer                     │
│            (tests/integration/*_test.go)            │
│         • Real Docker operations                     │
│         • End-to-end CLI execution                   │
│         • Container lifecycle verification           │
└─────────────────────────────────────────────────────┘
```

## Test Organization Strategy

### 1. Unit Tests - Command Layer (`cmd/*_test.go`)

**Location:** Same package as command files
**Purpose:** Test command-level logic without Docker dependencies

**Techniques:**
- Use `cobra.Command.SetOut()` to capture output
- Use `cobra.Command.SetArgs()` to inject arguments
- Mock Docker operations via interfaces
- Test validation and error handling

**Example Structure:**
```go
func TestSetupCommand_MissingNameFlag(t *testing.T) {
    cmd := setupCmd
    b := new(bytes.Buffer)
    cmd.SetOut(b)
    cmd.SetErr(b)
    cmd.SetArgs([]string{})

    err := cmd.Execute()
    assert.Error(t, err)
    assert.Contains(t, b.String(), "Missing required flag")
}
```

### 2. Unit Tests - Business Logic Layer (`internal/*_test.go`)

**Location:** Alongside internal packages
**Purpose:** Test business logic with mocked external dependencies

**Techniques:**
- Define `DockerClient` interface for Docker operations
- Create mock implementations for testing
- Test prerequisites validation independently
- Use table-driven tests for comprehensive coverage

**Example Interface:**
```go
type DockerClient interface {
    BuildImage(ctx context.Context, config BuildConfig) error
    RunContainer(ctx context.Context, config RunConfig) (string, error)
    ImageExists(ctx context.Context, image string) (bool, error)
}
```

### 3. Integration Tests (`tests/integration/*_test.go`)

**Location:** Separate `tests/integration/` directory
**Purpose:** Verify end-to-end behavior with real Docker

**Techniques:**
- Execute actual CLI binary via `exec.Command`
- Verify Docker containers/images exist
- Implement cleanup with `t.Cleanup()` or `defer`
- Use test fixtures for repeatable environments

**Example Structure:**
```go
func TestSetup_SuccessfulBuild(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Build CLI binary
    buildCLI(t)

    // Run setup command
    output := runCommand(t, "setup", "--name", "test-env")

    // Verify image exists
    assert.True(t, dockerImageExists(t, "spinner:test-env"))

    t.Cleanup(func() {
        removeDockerImage(t, "spinner:test-env")
    })
}
```

## Refactoring Requirements

### 1. Dependency Injection for Docker Operations

**Current Pattern:**
```go
func BuildImage(config BuildConfig) error {
    // Direct Docker CLI calls
    cmd := exec.Command("docker", "build", ...)
    return cmd.Run()
}
```

**Refactored Pattern:**
```go
type DockerClient interface {
    BuildImage(ctx context.Context, config BuildConfig) error
}

type RealDockerClient struct{}

func (c *RealDockerClient) BuildImage(ctx context.Context, config BuildConfig) error {
    cmd := exec.Command("docker", "build", ...)
    return cmd.Run()
}

// Commands accept client interface
func runSetup(cmd *cobra.Command, args []string, client DockerClient) error {
    return client.BuildImage(ctx, config)
}
```

### 2. Command Constructor Functions

**Enable testable command creation:**
```go
func NewSetupCommand(client DockerClient) *cobra.Command {
    return &cobra.Command{
        Use: "setup",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runSetup(cmd, args, client)
        },
    }
}
```

### 3. Test Utilities Organization

**Structure:**
```
tests/
├── testutil/
│   ├── docker.go       # Docker test helpers
│   ├── cli.go          # CLI execution helpers
│   ├── fixtures.go     # Test data and fixtures
│   └── assertions.go   # Custom assertions
└── integration/
    ├── setup_test.go
    └── spin_test.go
```

**Utilities:**
- `buildCLI(t)` - Build binary for integration tests
- `runCommand(t, args...)` - Execute CLI and capture output
- `dockerImageExists(t, image)` - Check image existence
- `dockerContainerRunning(t, name)` - Check container status
- `cleanup(t, resources...)` - Resource cleanup

## Migration Strategy

### Phase 1: Infrastructure Setup
1. Create test utility package (`tests/testutil/`)
2. Define Docker client interface (`internal/docker/client.go`)
3. Create mock Docker client (`internal/docker/mock_client_test.go`)
4. Set up integration test skeleton

### Phase 2: Setup Command Tests
1. Unit tests for setup command validation (`cmd/setup_test.go`)
2. Unit tests for Docker build operations (`internal/docker/build_test.go`)
3. Integration tests for setup command (`tests/integration/setup_test.go`)
4. Validate against bash test parity

### Phase 3: Spin Command Tests
1. Unit tests for spin command validation (`cmd/spin_test.go`)
2. Unit tests for Docker run operations (`internal/docker/run_test.go`)
3. Integration tests for spin command (`tests/integration/spin_test.go`)
4. Validate against bash test parity

### Phase 4: Validation & Cleanup
1. Run all Go tests and verify coverage
2. Run bash tests alongside for validation
3. Document any gaps or additional tests added
4. Update CI/CD to run Go tests
5. Deprecate bash tests (keep for reference)

## Test Naming Conventions

**Unit Tests:**
- `Test<Package>_<Scenario>` (e.g., `TestSetupCommand_MissingNameFlag`)
- `Test<Function>_<Scenario>` (e.g., `TestBuildImage_SuccessfulBuild`)

**Integration Tests:**
- `TestSetup_<Scenario>` (e.g., `TestSetup_SuccessfulBuild`)
- `TestSpin_<Scenario>` (e.g., `TestSpin_MissingImageFlag`)

## Table-Driven Test Pattern

For scenarios with multiple variations:

```go
func TestSetupCommand_Validation(t *testing.T) {
    tests := []struct {
        name      string
        args      []string
        wantError bool
        errorMsg  string
    }{
        {
            name:      "missing name flag",
            args:      []string{},
            wantError: true,
            errorMsg:  "Missing required flag",
        },
        {
            name:      "mutually exclusive flags",
            args:      []string{"--name", "test", "--base-image", "ubuntu", "--dockerfile", "Dockerfile"},
            wantError: true,
            errorMsg:  "mutually exclusive",
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Test Isolation & Cleanup

**Principles:**
1. Each test should be independent and isolated
2. Tests should not depend on execution order
3. Cleanup should always run, even on test failure
4. Use unique names for Docker resources per test

**Implementation:**
```go
func TestIntegration_Setup(t *testing.T) {
    // Generate unique test name
    testID := fmt.Sprintf("test-%d", time.Now().Unix())
    imageName := fmt.Sprintf("spinner:%s", testID)

    // Ensure cleanup runs
    t.Cleanup(func() {
        removeDockerImage(t, imageName)
    })

    // Test implementation
    // ...
}
```

## CI/CD Integration

**Test Execution:**
```bash
# Unit tests only (fast)
go test -short ./...

# All tests including integration
go test ./...

# With coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**GitHub Actions Example:**
```yaml
- name: Run unit tests
  run: go test -short -v ./...

- name: Run integration tests
  run: go test -v ./tests/integration/...
  env:
    DOCKER_BUILDKIT: 1
```

## Trade-offs & Decisions

| Decision | Rationale |
|----------|-----------|
| Hybrid testing approach | Balance speed (unit) with confidence (integration) |
| Interface-based mocking | Cleaner than exec mocking, follows Go best practices |
| Keep integration tests separate | Clear separation of concerns, can skip with `-short` |
| Table-driven tests | Comprehensive coverage with minimal code duplication |
| Maintain bash tests initially | Safety net during migration, validate equivalence |
| Use testify/assert | Standard library testing is verbose, testify improves readability |

## Dependencies

**Testing Libraries:**
- `github.com/stretchr/testify` - Assertions and mocking
- Standard library `testing` package
- No additional test frameworks needed

**Justification:**
- testify is widely adopted in Go community
- Provides cleaner assertions and mock generation
- Minimal dependency footprint
