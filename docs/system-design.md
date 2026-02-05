# System Design & Architecture

## Overview

Spinner is designed around a **provider abstraction pattern** that decouples the command layer from specific execution
backends. This architecture enables supporting multiple sandbox environments (Docker, VMs, cloud instances, Kubernetes)
without modifying core application logic.

## Provider Abstraction

### Core Concept

The `Provider` interface defines a backend-agnostic contract for managing execution environments. All commands interact
with this interface rather than directly with Docker, VMs, or other backends.

**Location:** `internal/provider/provider.go`

### Provider Interface

```go
type Provider interface {
// Provisioning
Setup(ctx context.Context, config SetupConfig) error

// Lifecycle Management
Create(ctx context.Context, config CreateConfig) error
Start(ctx context.Context, name string) error
Restart(ctx context.Context, name string) error
Stop(ctx context.Context, name string) error
Remove(ctx context.Context, name string) error

// Observability
Logs(ctx context.Context, name string) (io.ReadCloser, error)
Status(ctx context.Context, name string) (InstanceStatus, error)
InstanceName(config CreateConfig) string
}
```

### Configuration Pattern

Provider configurations use `Options map[string]string` for backend-specific parameters:

```go
type SetupConfig struct {
   Name    string            // Universal: environment name
   Options map[string]string // Backend-specific (e.g., "base-image", "dockerfile")
}

type CreateConfig struct {
   Repo          string // Universal fields
   Prompt        string
   Branch        string
   MaxIterations int
   Options       map[string]string // Backend-specific (e.g., "image")
}
```

This design allows each provider to define and parse its own options without coupling the command layer to
backend-specific types.

## Directory Structure

```
/
├── cmd/                           # CLI commands
│   ├── constructors.go            # Provider-agnostic command constructors
│   ├── setup.go                   # Setup command (wires Docker provider)
│   ├── spin.go                    # Spin command (wires Docker provider)
│   └── *_test.go                  # Tests using MockProvider
│
├── internal/
│   ├── provider/                  # Backend-agnostic abstractions
│   │   ├── provider.go            # Provider interface + types
│   │   └── mock_provider.go       # Mock for testing
│   │
│   ├── docker/                    # Docker provider implementation
│   │   ├── docker_provider.go     # Implements Provider interface
│   │   ├── client.go              # DockerClient interface
│   │   ├── mock_client.go         # Mock client for testing
│   │   ├── run.go                 # Container run logic
│   │   ├── build.go               # Image build logic
│   │   ├── dockerfile.go          # Dockerfile generation
│   │   └── *_test.go              # Docker-specific tests
│   │
│   ├── agent/                     # AI agent integration (provider-agnostic)
│   ├── exec/                      # Execution logic (provider-agnostic)
│   └── prerequisites/             # Environment validation
│
└── docs/                          # Documentation
```

## Separation of Concerns

### Provider-Agnostic Code (`cmd/`, `internal/agent/`, `internal/exec/`)

The command layer and execution logic operate on the Provider interface without knowing implementation details:

```go
// cmd/constructors.go
func NewSpinCommand(p provider.Provider) *cobra.Command {
   // Works with any Provider implementation
   status, err := p.Status(ctx, name)
   if err == nil && status == provider.Running {
	   p.Start(ctx, name)
   } else {
	   p.Create(ctx, config)
   }
}
```

**Responsibilities:**

- Repository and git validation
- User input processing
- Iteration loop management
- Log display and formatting
- Error handling and reporting

### Provider-Specific Code (`internal/docker/`, future: `internal/vm/`, etc.)

Each provider package contains all backend-specific logic:

**Docker Provider Responsibilities:**

- Container name generation
- Dockerfile creation and templating
- Docker CLI invocation (`docker build`, `docker run`, etc.)
- npmrc detection and volume mounting
- Container status mapping to InstanceStatus
- Docker-specific error handling

**Future Providers (e.g., VM, GCP, K8s):**

- Each would implement Provider interface in its own package
- Define its own internal client interface and types
- Parse Options map according to its needs
- No changes required to command layer

## Dependency Injection

Commands receive providers via constructor injection:

```go
// Production wiring (cmd/setup.go, cmd/spin.go)
var setupCmd = NewSetupCommand(
    docker.NewDockerProvider(docker.NewRealDockerClient())
)

// Test wiring (cmd/*_test.go)
mockProvider := provider.NewMockProvider(t)
cmd := NewSetupCommand(mockProvider)
```

Benefits:

- Easy testing with MockProvider
- Swap providers without changing commands
- Clear dependency boundaries

## Docker Provider Architecture

### Two-Layer Design

```
Command Layer (cmd/)
       ↓
Provider Interface (internal/provider/)
       ↓
DockerProvider (internal/docker/docker_provider.go)
       ↓
DockerClient Interface (internal/docker/client.go)
       ↓
RealDockerClient / MockDockerClient
```

**Why Two Layers?**

1. **Provider Layer:** Backend abstraction (Docker vs VM vs cloud)
2. **Client Layer:** Enables testing Docker-specific logic without Docker CLI

### DockerClient Interface

```go
type DockerClient interface {
   BuildImage(ctx context.Context, config BuildConfig) error
   RunContainer(ctx context.Context, args []string, name string) error
   ImageExists(ctx context.Context, image string) (bool, error)
   ContainerExists(ctx context.Context, name string) (bool, ContainerStatus, error)
   StartContainer(ctx context.Context, name string) error
   StopContainer(ctx context.Context, name string) error
   RemoveContainer(ctx context.Context, name string) error
   LogsContainer(ctx context.Context, name string) (io.ReadCloser, error)
}
```

**Implementations:**

- `RealDockerClient`: Executes actual `docker` CLI commands
- `MockDockerClient`: Testify mock for unit tests

## Testing Strategy

### Command Tests

- Use `MockProvider` from `internal/provider/mock_provider.go`
- No Docker dependency
- Fast, isolated unit tests

### Provider Tests

- Use `MockDockerClient` from `internal/docker/mock_client.go`
- Test Docker-specific logic without Docker runtime
- Example: `internal/docker/docker_provider_test.go`

### Integration Tests

- Use real Docker client (future work)
- Verify end-to-end functionality

## Adding a New Provider

To add a new execution backend (e.g., VM, Kubernetes, cloud instance):

1. **Create provider package:** `internal/newbackend/`

2. **Implement Provider interface:**
   ```go
   type NewBackendProvider struct {
       client NewBackendClient
   }

   func (p *NewBackendProvider) Setup(ctx context.Context, config provider.SetupConfig) error {
       // Parse config.Options for backend-specific fields
       // Provision environment
   }

   // Implement remaining Provider methods...
   ```

3. **Define internal client interface:**
   ```go
   type NewBackendClient interface {
       // Backend-specific operations
   }
   ```

4. **Create real and mock implementations:**
    - `RealNewBackendClient` - actual backend interaction
    - `MockNewBackendClient` - for testing

5. **Wire into commands:**
   ```go
   // cmd/setup.go
   var setupCmd = NewSetupCommand(
       newbackend.NewProvider(newbackend.NewRealClient())
   )
   ```

6. **Write tests:**
    - Provider tests using MockClient
    - Command tests using MockProvider (no changes needed)

**No changes required to:**

- Command layer (`cmd/`)
- Provider-agnostic packages (`internal/agent/`, `internal/exec/`)
- Provider interface definition

## Design Principles

### Minimal Coupling

- Commands depend only on Provider interface
- Provider implementations hidden in packages
- Each provider manages its own types and logic

### Options Pattern

- `map[string]string` for backend-specific configuration
- No enum variants or type switches in command layer
- Each provider interprets its own options

### Testability

- Every layer has a mock implementation
- Tests run without external dependencies
- Clear boundaries enable isolated testing

### Explicit Lifecycle

- Deterministic instance naming via `InstanceName()`
- Complete state management: Create → Start → Stop → Remove
- Status checking before operations

## Future Extensions

The architecture supports future enhancements:

- **Multiple providers:** Allow users to choose backend via CLI flag
- **Provider registry:** Dynamic provider selection based on configuration
- **Hybrid deployments:** Different providers for different environments
- **Provider plugins:** Load external provider implementations
