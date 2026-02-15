# Code Review — 2026-02-15

## Code Quality

### Unused / Dead Code

- [ ] `internal/version/version.go:23` — `Full()` is exported but never called. Remove or unexport.
- [ ] `internal/backend/gcp/state.go:38` — `writeState()` only called from tests, never production code. Move to test helper or remove.
- [ ] `internal/prerequisites/prerequisites.go:8-10` — `environmentVariableError.Variable` field is set but never read; only `.Error()` is called. Remove the field.
- [ ] `internal/tui/watch.go:901` — `WatchUI.Context()` defined but never called. Remove.
- [ ] `internal/backend/docker/events.go:84` — `defaultLogStreamOptions()` only used in test files. Move to test file.

### Swallowed Errors

- [ ] `internal/backend/docker/docker_provider.go:322` — `_ = fmt.Errorf(...)` creates and discards a `MAX_ITERATIONS` parse error. Surface or log it.
- [ ] `internal/backend/gcp/gcp_provider.go:443` — Same swallowed `MAX_ITERATIONS` parse error in GCP backend.

### Significant Code Duplication

- [ ] `internal/backend/docker/run.go:69-80` + GCP equivalent — `extractRepoName()` / `sanitizeComponent()` duplicated across backends. Extract to shared util.
- [ ] `internal/backend/gcp/gcp_provider.go` lines 42-54 vs 154-166 — Machine type defaulting + disk size parsing copy-pasted between `Setup()` and `Create()`. Consolidate.
- [ ] `internal/backend/gcp/client.go` lines 433-456 vs 459-482 — `ReadObject()` and `ReadObjectRange()` contain identical manual read loops. Use `io.ReadAll` or shared helper.

### Broken Abstractions

- [ ] `internal/backend/docker/docker_provider.go:404-472` — `getDockerContainerID`, `getDockerImageID`, and `getContainerEnvVars` bypass the `Client` interface and shell out via `exec.Command`. Breaks provider/client abstraction and makes these paths untestable with the mock. Route through the client interface.

### Inconsistent Patterns

- [ ] `internal/backend/docker/docker_provider.go:406,421` — Hardcoded `5*1000000000` (raw nanoseconds) instead of `5*time.Second`. Fix to use `time` constants.
- [ ] `internal/backend/docker/docker_provider.go:192` + `internal/backend/gcp/gcp_provider.go:317` — Both `Restart()` methods call `Start()` with empty `provider.CreateConfig{}`, clobbering previous prompt/model/config in state. Preserve existing config on restart.

### Overly Complex Functions

- [ ] `internal/tui/watch.go:84-194` — `NewWatchUI()` spans 110 lines mixing widget construction, layout, state init, and keyboard handlers. Split into focused helpers.
- [ ] `internal/backend/gcp/gcp_provider.go:38-130` — `Setup()` is ~93 lines mixing image checking, file I/O, env mutation (`os.Setenv` global side effect on line 103), and orchestration. Break apart and eliminate `os.Setenv` side effect.

---

## Security

### Won't Fix

The following were reviewed and dismissed:

| Finding | Reason |
|---------|--------|
| Command injection via container/instance name in shell commands (`docker_provider.go:409,424,439`) | User controls both host and VM. Low practical risk. |
| Command injection via bake script template (`gcp/startup.go:28-44`) | User controls both host and VM. Low practical risk. |
| Passwordless sudo + `--dangerously-skip-permissions` | This is the intended design — the container is the sandbox. |
| Weak URL validation (`cmd/helpers.go:63-67`) | User-controlled input in a local CLI tool. Not an attack surface. |
| Unrestricted container networking | Outbound access is required for the agent to function. |
| GCP metadata token exposure (`gcp_provider.go:179-183`) | Will be addressed by the secret store spec. |

### Fix

- [ ] `internal/exec/config.go:36-41` — No upper bound on `max-iterations`. Add a sane ceiling (e.g. 500 or 1000) to prevent unbounded cost/resource consumption.
- [ ] `internal/backend/docker/run.go:122-152` — Env file newline injection. Values containing `\n` can inject arbitrary env vars, bypassing the reserved-key check. Sanitize or reject values with embedded newlines.
- [ ] `internal/backend/docker/docker_provider.go:155,217` — Path traversal via container name in state directory. `containerName` used directly in `filepath.Join`. Validate that the resolved path stays within `~/.spinner/`.
- [ ] `internal/backend/docker/templates/docker/extending.template:10,35` — Piped `curl | bash` for Node.js and Claude CLI with no version pinning or checksum verification. Pin versions and verify checksums.
- [ ] `internal/backend/gcp/image.go:99`, `gcp/gcp_provider.go:219` — GCP VMs created with `ExternalIP: true` by default. Make external IP opt-in or off by default for runtime VMs.
