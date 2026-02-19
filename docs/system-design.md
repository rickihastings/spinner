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
    Options map[string]string // Backend-specific (e.g., "dockerfile" for Docker)
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
│   ├── factory.go                 # Provider factory wiring (Docker + GCP)
│   ├── helpers.go                 # Shared flag helpers and validation
│   ├── setup.go                   # Setup command
│   ├── spin.go                    # Spin command
│   ├── watch.go                   # Watch command
│   ├── exec.go                    # Exec command (runs inside containers/VMs)
│   └── *_test.go                  # Tests using MockProvider
│
├── internal/
│   ├── provider/                  # Backend-agnostic abstractions
│   │   ├── provider.go            # Provider interface + types
│   │   ├── factory.go             # Provider factory (backend registry)
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
│   ├── gcp/                       # GCP Compute Engine provider
│   │   ├── gcp_provider.go        # Implements Provider interface
│   │   ├── client.go              # GCP Client interface + RealGCPClient
│   │   ├── mock_client.go         # Mock client for testing
│   │   ├── image.go               # Image baking (temp VM → custom image)
│   │   ├── instance.go            # Instance naming, status mapping
│   │   ├── types.go               # GCP-specific types (InstanceConfig, VMStatus)
│   │   ├── logs.go                # GCS log streaming (read/watch)
│   │   ├── metrics.go             # System metrics (CPU/memory via GCS state)
│   │   ├── state.go               # GCS state persistence (read/write)
│   │   ├── startup.go             # Startup script template loading
│   │   └── *_test.go              # GCP-specific tests
│   │
│   ├── logs/                      # Log streaming utilities
│   │   └── gcs_sink.go            # GCSSink: buffered io.Writer → GCS
│   │
│   ├── agent/                     # AI agent integration (provider-agnostic)
│   ├── exec/                      # Execution logic (provider-agnostic)
│   └── prerequisites/             # Environment validation
│
├── templates/scripts/             # Shell script templates
│   ├── startup.sh                 # Common startup (clone repo, run exec)
│   ├── gcp_runtime.sh             # GCP VM boot: metadata → env, state restore
│   └── gcp_bake.sh                # GCP image bake: install tooling
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

### Provider-Specific Code (`internal/docker/`, `internal/gcp/`)

Each provider package contains all backend-specific logic:

**Docker Provider Responsibilities:**

- Container name generation
- Dockerfile creation and templating
- Docker CLI invocation (`docker build`, `docker run`, etc.)
- npmrc detection and volume mounting
- Container status mapping to InstanceStatus
- Docker-specific error handling

**GCP Provider Responsibilities:**

- VM instance lifecycle management via Compute Engine SDK
- Custom image baking (temporary VM → install tooling → create image)
- Deterministic instance naming (GCP 63-char constraints)
- Log streaming via GCS (buffered writes from VM, polled reads from control plane)
- State persistence via GCS (synced after each state change in exec loop)
- CPU/memory metrics via GCS state file
- VM status mapping to InstanceStatus

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

## GCP Provider Architecture

### Two-Layer Design

```
Command Layer (cmd/)
       ↓
Provider Interface (internal/provider/)
       ↓
GCPProvider (internal/gcp/gcp_provider.go)
       ↓
Client Interface (internal/gcp/client.go)
       ↓
RealGCPClient / MockGCPClient
```

### GCP Client Interface

The `Client` interface wraps GCP SDK clients (Compute, Storage) for testability:

- **Instance ops:** CreateInstance, GetInstance, StartInstance, StopInstance, ResetInstance, DeleteInstance
- **Image ops:** CreateImage, GetImage, DeleteImage
- **Storage ops:** WriteObject, ReadObject, ReadObjectRange, ObjectSize, ObjectExists
- **Diagnostics:** GetSerialPortOutput

### State Persistence

GCP state persists to GCS at `gs://{bucket}/{instance}/state.json`:

- **Download on boot:** `gcp_runtime.sh` restores state from GCS before running `spinner exec`
- **Upload after each state change:** The exec loop syncs state to GCS after each local `SaveState()` call
- **Env-driven activation:** Sync is enabled when `SPINNER_STATE_BUCKET` and `SPINNER_INSTANCE_NAME` are set on a GCE VM

### Log Streaming

Two-tier approach using GCS as the transport:

- **VM side:** `GCSSink` (buffered `io.Writer`) flushes logs to GCS every 2 seconds
- **Control plane:** `WatchLogs` polls GCS with range reads to stream new content
