## 1.0 Implement deterministic container naming

- [ ] 1.1 Create `sanitizeComponent()` helper function in `utils/docker.ts`
- [ ] 1.2 Create `extractRepoName()` helper function in `utils/docker.ts`
- [ ] 1.3 Update `generateContainerName()` signature to accept full `SpinConfig`
- [ ] 1.4 Implement deterministic naming: `{image}-{repo}` or `{image}-{repo}-{branch}`
- [ ] 1.5 Update Spin.tsx to pass full config to `generateContainerName()`
- [ ] 1.6 Update help text to explain deterministic naming behavior
- [ ] 1.7 Add tests for deterministic naming (image+repo, image+repo+branch, sanitization)
- [ ] 1.8 Update existing tests that rely on timestamp-based names
- [ ] 1.9 Verify all tests pass

## 2.0 Implement container reuse logic

- [ ] 2.1 Create `checkContainerExists()` function returning 'running' | 'stopped' | 'none'
- [ ] 2.2 Create `restartContainer()` function to restart stopped containers
- [ ] 2.3 Add TypeScript interfaces (ContainerStatus, ReuseResult)
- [ ] 2.4 Update Spin component to check existence before creation
- [ ] 2.5 Implement reuse: running → use as-is, stopped → restart, none → create
- [ ] 2.6 Update CLI output to distinguish created/reused/restarted states
- [ ] 2.7 Update management instructions output to mention reuse behavior
- [ ] 2.8 Add tests for reusing running container
- [ ] 2.9 Add tests for restarting stopped container
- [ ] 2.10 Verify all tests pass

## 3.0 Implement --recreate flag

- [ ] 3.1 Add `--recreate` boolean flag to App.tsx spin command
- [ ] 3.2 Add `recreate?: boolean` to SpinProps interface
- [ ] 3.3 Create `removeContainer()` function to force-remove existing containers
- [ ] 3.4 Implement recreate logic: remove existing then create fresh
- [ ] 3.5 Update help text to document `--recreate` behavior
- [ ] 3.6 Add test for `--recreate` flag removes and recreates container
- [ ] 3.7 Verify all tests pass
