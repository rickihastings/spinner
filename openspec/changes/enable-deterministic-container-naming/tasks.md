## 1. Name Generation Updates

- [ ] 1.1 Create `sanitizeComponent()` helper function in `utils/docker.ts`
- [ ] 1.2 Create `extractRepoName()` helper function in `utils/docker.ts`
- [ ] 1.3 Modify `generateContainerName()` to use deterministic naming based on image + repo + branch
- [ ] 1.4 Update `generateContainerName()` function signature to accept full `SpinConfig` instead of just repo string

## 2. Container Reuse Logic

- [ ] 2.1 Create `checkContainerExists()` function that returns 'running' | 'stopped' | 'none'
- [ ] 2.2 Create `restartContainer()` function to restart stopped containers
- [ ] 2.3 Create `removeContainer()` function to force-remove existing containers
- [ ] 2.4 Update TypeScript interfaces to support reuse scenarios (ReuseResult, ContainerStatus, etc.)

## 3. Spin Command Updates

- [ ] 3.1 Add `recreate?: boolean` to `SpinProps` interface in `Spin.tsx`
- [ ] 3.2 Update Spin component to check container existence before creation
- [ ] 3.3 Implement reuse logic: handle running, stopped, and non-existent containers
- [ ] 3.4 Implement `--recreate` flag behavior: remove existing and create fresh
- [ ] 3.5 Update CLI output messages to distinguish between created/reused/restarted states

## 4. CLI Flag Addition

- [ ] 4.1 Add `--recreate` boolean flag to App.tsx spin command definition
- [ ] 4.2 Pass `recreate` prop to Spin component
- [ ] 4.3 Update help text to document `--recreate` flag behavior

## 5. Testing

- [ ] 5.1 Create test for deterministic name generation with image + repo
- [ ] 5.2 Create test for deterministic name generation with image + repo + branch
- [ ] 5.3 Create test for name sanitization (special characters, colons, slashes)
- [ ] 5.4 Create test for reusing running container
- [ ] 5.5 Create test for restarting stopped container
- [ ] 5.6 Create test for `--recreate` flag removes and recreates container
- [ ] 5.7 Update existing tests that rely on unique timestamp-based names

## 6. Documentation

- [ ] 6.1 Update CLI help text to explain deterministic naming behavior
- [ ] 6.2 Update management instructions output to mention reuse behavior
- [ ] 6.3 Add examples of deterministic naming to help text
