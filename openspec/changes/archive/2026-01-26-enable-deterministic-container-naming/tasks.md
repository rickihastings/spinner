## 1.0 Implement deterministic container naming

- [x] 1.1 Create `sanitizeComponent()` helper function in `utils/docker.ts`
- [x] 1.2 Create `extractRepoName()` helper function in `utils/docker.ts`
- [x] 1.3 Update `generateContainerName()` signature to accept full `SpinConfig`
- [x] 1.4 Implement deterministic naming: `{image}-{repo}` or `{image}-{repo}-{branch}`
- [x] 1.5 Update Spin.tsx to pass full config to `generateContainerName()`
- [x] 1.6 Update help text to explain deterministic naming behavior
- [x] 1.7 Add tests for deterministic naming (image+repo, image+repo+branch, sanitization)
- [x] 1.8 Update existing tests that rely on timestamp-based names
- [x] 1.9 Verify all tests pass

## 2.0 Implement container reuse logic

- [x] 2.1 Create `checkContainerExists()` function returning 'running' | 'stopped' | 'none'
- [x] 2.2 Create `restartContainer()` function to restart stopped containers
- [x] 2.3 Add TypeScript interfaces (ContainerStatus, ReuseResult)
- [x] 2.4 Update Spin component to check existence before creation
- [x] 2.5 Implement reuse: running → use as-is, stopped → restart, none → create
- [x] 2.6 Update CLI output to distinguish created/reused/restarted states
- [x] 2.7 Update management instructions output to mention reuse behavior
- [x] 2.8 Add tests for reusing running container
- [x] 2.9 Add tests for restarting stopped container
- [x] 2.10 Verify all tests pass

## 3.0 Implement --recreate flag

- [x] 3.1 Add `--recreate` boolean flag to App.tsx spin command
- [x] 3.2 Add `recreate?: boolean` to SpinProps interface
- [x] 3.3 Create `removeContainer()` function to force-remove existing containers
- [x] 3.4 Implement recreate logic: remove existing then create fresh
- [x] 3.5 Update help text to document `--recreate` behavior
- [x] 3.6 Add test for `--recreate` flag removes and recreates container
- [x] 3.7 Verify all tests pass
