# Design: Docker SDK Migration

## Context

The `internal/docker` package currently implements all Docker operations by spawning CLI commands via `os/exec.CommandContext`. While functional, this approach cannot support real-time event consumption needed for upcoming features like live log streaming, build progress reporting, and container health monitoring.

The Docker Go SDK provides native access to the Docker Engine API with full event streaming support.

## Goals / Non-Goals

**Goals:**
- Replace CLI invocations with SDK calls while maintaining identical behavior
- Enable real-time event streaming for logs and build progress
- Keep all changes isolated to `internal/docker/` package
- Maintain full test coverage and passing integration tests

**Non-Goals:**
- Changing the `DockerClient` interface signature (except adding new methods)
- Modifying callers in `cmd/` package
- Adding new user-facing features (this is infrastructure only)
- Supporting Docker Compose or multi-container orchestration

## Technical Implementation Plan

### Component Map

| File | Change | Action |
|------|--------|--------|
| `internal/docker/client.go` | Replace exec.Command calls with SDK client calls | modify |
| `internal/docker/sdk.go` | New file: SDK client initialization and helpers | create |
| `internal/docker/events.go` | New file: Event types for streaming | create |
| `internal/docker/mock_client.go` | Add new mock methods for streaming | modify |
| `internal/docker/client_test.go` | Update tests for new implementation | modify |
| `go.mod` | Add docker SDK dependency | modify |

### SDK Method Mapping

| Current CLI Command | SDK Replacement |
|---------------------|-----------------|
| `docker build` | `client.ImageBuild()` |
| `docker run -d` | `client.ContainerCreate()` + `client.ContainerStart()` |
| `docker image inspect` | `client.ImageInspectWithRaw()` |
| `docker inspect` | `client.ContainerInspect()` |
| `docker rm -f` | `client.ContainerRemove()` with Force option |
| `docker start` | `client.ContainerStart()` |
| `docker logs` | `client.ContainerLogs()` with stream |

### Approach

**Phase 1: Foundation**
1. Add Docker SDK dependency
2. Create SDK client initialization wrapper with connection handling
3. Add event types for progress/log streaming

**Phase 2: Container Operations (Low Risk)**
Migrate simpler operations first:
1. `ImageExists` - Simple inspect call
2. `ContainerExists` - Container inspect for status
3. `RemoveContainer` - Container remove with force
4. `RestartContainer` - Container start

**Phase 3: Run Operation (Medium Risk)**
1. `RunContainer` - Convert to ContainerCreate + ContainerStart
2. Handle volume mounts via SDK's mount types
3. Handle environment variables via SDK's container config
4. Maintain log directory creation logic

**Phase 4: Build Operation (Highest Risk)**
1. `BuildImage` - Convert to ImageBuild with tar context
2. Handle Dockerfile generation in memory
3. Stream build output for progress
4. Handle user Dockerfile building

**Phase 5: New Streaming Capabilities**
1. Add `StreamContainerLogs(ctx, name) (<-chan LogEvent, error)`
2. Add `StreamBuildProgress(ctx, config) (<-chan BuildEvent, error)` (optional)

### Patterns to Follow

See `client.go:46-136` for current BuildImage pattern - maintain same high-level flow:
1. Create temp build context
2. Handle user Dockerfile if provided
3. Generate extending Dockerfile
4. Copy build files
5. Build spinner binary
6. Build final image

The SDK version will use the same flow but with `ImageBuild` instead of exec.

### Key Decisions

1. **SDK Client Lifecycle**: Create new client per operation vs reuse
   - Decision: Create lazily on first use, store in RealDockerClient struct
   - Rationale: Avoids connection issues, minimal overhead

2. **Build Context**: Use tar archive (SDK requirement) vs temp directory
   - Decision: Build tar in memory from temp directory
   - Rationale: Keeps existing file copying logic, SDK requires tar

3. **Error Types**: Return raw SDK errors vs wrap them
   - Decision: Wrap with context, maintain current error message style
   - Rationale: Consistent error UX for users

4. **Progress Streaming**: Blocking channel vs callback pattern
   - Decision: Channel-based streaming (`<-chan Event`)
   - Rationale: Idiomatic Go, composable with select

### SDK Client Initialization

```go
type RealDockerClient struct {
    cli *client.Client
    mu  sync.Mutex
}

func (c *RealDockerClient) getClient(ctx context.Context) (*client.Client, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.cli == nil {
        cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
        if err != nil {
            return nil, fmt.Errorf("failed to create Docker client: %w", err)
        }
        c.cli = cli
    }
    return c.cli, nil
}
```

### Container Create Example

```go
func (c *RealDockerClient) RunContainer(ctx context.Context, args []string, containerName string) (ContainerResult, error) {
    cli, err := c.getClient(ctx)
    if err != nil {
        return ContainerResult{Success: false, Error: err.Error()}, err
    }

    // Parse args into ContainerConfig and HostConfig
    config := &container.Config{
        Image: imageName,
        Env:   envVars,
    }
    hostConfig := &container.HostConfig{
        Mounts: mounts,
    }

    resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
    if err != nil {
        return ContainerResult{Success: false, Error: err.Error()}, err
    }

    if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
        return ContainerResult{Success: false, Error: err.Error()}, err
    }

    return ContainerResult{Success: true, ContainerName: containerName}, nil
}
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| SDK version compatibility | Medium | Pin SDK version, use API version negotiation |
| Build context tar creation | Low | Test thoroughly, same files as current |
| Connection handling | Low | Lazy init with proper error handling |
| Platform differences in SDK | Low | SDK handles this internally |

## Open Questions

1. Should we add `Close()` method to DockerClient interface for cleanup?
   - Recommendation: Yes, for proper resource management

2. Should streaming methods be on main interface or separate interface?
   - Recommendation: Add to main interface, mock can return closed channel

## Testing Strategy

1. **Unit Tests**: Mock SDK client using interface
2. **Integration Tests**: Existing tests should pass unchanged
3. **New Tests**: Add tests for streaming functionality
4. **Manual Testing**: Verify build and run operations work identically
