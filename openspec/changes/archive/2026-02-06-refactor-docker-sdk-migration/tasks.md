# Tasks: Docker SDK Migration

## 1.0 Add Docker SDK dependency and create foundation

- [x] 1.1 Add `github.com/docker/docker` SDK to go.mod
- [x] 1.2 Create `internal/docker/sdk.go` with SDK client initialization wrapper
- [x] 1.3 Create `internal/docker/events.go` with event types for streaming
- [x] 1.4 Add tests for SDK client initialization
- [x] 1.5 Verify build succeeds with new dependency

## 2.0 Migrate simple container inspection operations

- [x] 2.1 Convert `ImageExists` to use `ImageInspectWithRaw`
- [x] 2.2 Convert `ContainerExists` to use `ContainerInspect`
- [x] 2.3 Update unit tests for inspection methods
- [x] 2.4 Run integration tests to verify behavior unchanged

## 3.0 Migrate container lifecycle operations

- [x] 3.1 Convert `RemoveContainer` to use `ContainerRemove` with Force option
- [x] 3.2 Convert `RestartContainer` to use `ContainerStart`
- [x] 3.3 Convert `VerifyContainerStatus` to use `ContainerInspect` and `ContainerLogs`
- [x] 3.4 Update unit tests for lifecycle methods
- [x] 3.5 Run integration tests to verify behavior unchanged

## 4.0 Migrate RunContainer operation

- [ ] 4.1 Create helper to convert current args slice to SDK ContainerConfig
- [ ] 4.2 Create helper to build HostConfig with volume mounts
- [ ] 4.3 Implement `RunContainer` using `ContainerCreate` + `ContainerStart`
- [ ] 4.4 Handle log retrieval on failure using SDK's `ContainerLogs`
- [ ] 4.5 Update unit tests for RunContainer
- [ ] 4.6 Run integration tests to verify spin command works

## 5.0 Migrate BuildImage operation

- [x] 5.1 Create helper to build tar archive from build context directory
- [x] 5.2 Implement user Dockerfile build step using `ImageBuild`
- [x] 5.3 Implement final image build using `ImageBuild` with generated Dockerfile
- [x] 5.4 Handle build output streaming and error detection
- [x] 5.5 Maintain Go binary cross-compilation step (keep exec for go build)
- [x] 5.6 Update unit tests for BuildImage
- [x] 5.7 Run integration tests to verify setup command works

## 6.0 Add streaming capabilities for future work

- [x] 6.1 Add `StreamContainerLogs` method to DockerClient interface
- [x] 6.2 Implement log streaming using SDK's `ContainerLogs` with Follow option
- [x] 6.3 Add mock implementation for `StreamContainerLogs`
- [x] 6.4 Add tests for log streaming functionality
- [x] 6.5 Document new streaming API in code comments

## 7.0 Final validation and cleanup

- [ ] 7.1 Run full integration test suite
- [ ] 7.2 Run unit tests with coverage report
- [ ] 7.3 Verify coverage is equivalent to pre-migration
- [ ] 7.4 Remove any dead code from CLI implementation
- [ ] 7.5 Update any relevant documentation
