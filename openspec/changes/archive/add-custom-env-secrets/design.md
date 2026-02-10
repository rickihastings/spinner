# Design: Add Custom Environment Variables / Secrets

## Docker Secret-Passing Options Analysis

Before choosing an implementation, we evaluated the available Docker mechanisms for passing secrets to
containers at runtime. Spinner uses `docker run` (standalone containers, not Swarm), so only mechanisms
compatible with `docker run` are viable.

### Option 1: Environment Variables (`-e KEY=VALUE`)

Current approach for `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN`.

| Property | Status |
|---|---|
| Visible in `docker inspect` | Yes (full value in `.Config.Env`) |
| Visible in `ps aux` | **Yes** (full value on command line) |
| Persisted in image layers | No |

**Pros:** Trivial to implement, universally supported, no temp files.
**Cons:** Secret values visible in host process listings (`ps auxww`). Any user with Docker socket
access sees them in `docker inspect`.

### Option 2: `--env-file` with Temporary File

Write secrets to a temp file, pass `--env-file /path/to/file`, delete after container starts.

| Property | Status |
|---|---|
| Visible in `docker inspect` | Yes (values appear as env vars) |
| Visible in `ps aux` | **No** (only file path on command line) |
| Persisted in image layers | No |

**Pros:** Keeps secret values out of process listings. Clean handling of many variables. Easy to
implement in Go (~15 lines: create temp file, write, pass flag, defer cleanup).
**Cons:** Secrets still visible in `docker inspect`. File exists on disk briefly (but with `0600`
permissions and immediate cleanup). Requires reliable cleanup on crashes.

### Option 3: Docker Secrets (Swarm Mode)

Secrets stored encrypted, mounted as tmpfs files at `/run/secrets/<name>`.

**Not viable.** Requires Docker Swarm mode. Only works with `docker service create`, not `docker run`.
Spinner's architecture is built around standalone containers.

### Option 4: Bind-Mounted Secret Files

Write secrets to host files, bind-mount read-only into container at `/run/secrets/<name>`.

| Property | Status |
|---|---|
| Visible in `docker inspect` | Mount path only (not contents) |
| Visible in `ps aux` | No |
| Persisted in image layers | No |

**Pros:** Best security of the viable options. Secret values hidden from both `ps` and `docker inspect`.
Follows the `/run/secrets/` convention. Read-only mount prevents container modification.
**Cons:** Higher complexity: must create files, mount them, clean up after container removal. Requires
the container entrypoint to read from files instead of env vars (or a bridging script). Secret files
persist on host until explicitly deleted.

### Option 5: BuildKit `--secret` (Build-Time Only)

**Not applicable.** Build-time only mechanism. Cannot pass runtime secrets.

### Recommendation

**Use `--env-file` with a temporary file (Option 2).**

Rationale:
- **Minimal change from current approach.** Secrets still arrive as environment variables inside the
  container, so startup scripts (`startup.sh`, `gcp_runtime.sh`) work unchanged. No entrypoint
  modifications needed.
- **Meaningful security improvement.** Removes secret values from `ps aux` output, which is the most
  commonly exploited leakage vector on shared hosts.
- **Simple to implement.** Write temp file with `0600` permissions, pass `--env-file`, `defer os.Remove()`.
  The cleanup is straightforward and follows Go conventions.
- **Consistent with GCP approach.** GCP uses instance metadata which is conceptually identical to
  environment variables (key-value pairs, accessible by the process). Using `--env-file` for Docker
  keeps the mental model the same across backends.

The bind-mounted files approach (Option 4) offers marginally better security (`docker inspect` doesn't
show values) but requires entrypoint changes, file lifecycle management, and breaks the simple "env var"
mental model. This complexity isn't justified for a developer CLI tool where the user owns the Docker
socket anyway. If stronger isolation is needed in the future, Option 4 or GCP Secret Manager can be
added as a follow-up without changing the `--env` CLI surface.

---

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|---|---|---|
| `cmd/helpers.go` | **modify** | Add `flagEnv` constant |
| `cmd/spin.go` | **modify** | Add `--env` flag (string slice), pass to `CreateConfig` |
| `internal/provider/provider.go` | **modify** | Add `EnvVars` field to `CreateConfig` |
| `internal/backend/docker/run.go` | **modify** | Add `EnvVars` to `SpinConfig`, write `--env-file` temp file |
| `internal/backend/docker/docker_provider.go` | **modify** | Pass `EnvVars` from `CreateConfig` to `SpinConfig` |
| `internal/backend/gcp/gcp_provider.go` | **modify** | Merge custom env vars into instance metadata |
| `templates/scripts/gcp_runtime.sh` | **modify** | Read custom env vars from metadata prefix `SPINNER_ENV_` |
| `internal/backend/docker/run_test.go` | **modify** | Test `--env-file` generation and cleanup |
| `internal/backend/gcp/gcp_provider_test.go` | **modify** | Test custom env vars in metadata |
| `cmd/spin_test.go` | **modify** | Test `--env` flag parsing and validation |

### Approach

#### CLI Surface

Add a repeatable `--env` flag to the `spin` command:

```bash
# Single variable
spinner spin --image my-env --repo ... --env NPM_TOKEN=abc123

# Multiple variables
spinner spin --image my-env --repo ... --env NPM_TOKEN=abc123 --env MY_API_KEY=xyz

# Combined with other flags
spinner spin --backend gcp --image my-env --repo ... --env NPM_TOKEN=abc123 --prompt "Fix bug"
```

The `--env` flag:
- Accepts `KEY=VALUE` format (validated: must contain `=`, key must be non-empty)
- Can be specified multiple times (Cobra `StringSliceVar`)
- Works with both Docker and GCP backends
- Is a general flag (not backend-specific), so no conditional validation needed

#### Provider Interface Change

Add `EnvVars` to `CreateConfig`:

```go
type CreateConfig struct {
    Repo          string
    Prompt        string
    Branch        string
    MaxIterations string
    Options       map[string]string
    EnvVars       map[string]string  // NEW: custom env vars from --env flag
}
```

Using `map[string]string` (not `[]string`) so that duplicate keys are resolved (last wins) and
backends can iterate cleanly. The `cmd/spin.go` parses the `KEY=VALUE` strings into the map.

#### Docker Backend

In `BuildDockerRunCommand`, write an env-file for all environment variables (both built-in and custom):

```go
// Write all env vars (built-in + custom) to a temp file
tmpFile, err := os.CreateTemp("", "spinner-env-")
if err != nil {
    return nil, fmt.Errorf("failed to create env file: %w", err)
}
defer os.Remove(tmpFile.Name())
os.Chmod(tmpFile.Name(), 0600)

// Write built-in vars
fmt.Fprintf(tmpFile, "GITHUB_TOKEN=%s\n", os.Getenv("GITHUB_TOKEN"))
fmt.Fprintf(tmpFile, "CLAUDE_CODE_OAUTH_TOKEN=%s\n", os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"))

// Write custom vars
for key, value := range config.EnvVars {
    fmt.Fprintf(tmpFile, "%s=%s\n", key, value)
}
tmpFile.Close()

// Use --env-file instead of individual -e flags
dockerArgs = append(dockerArgs, "--env-file", tmpFile.Name())
```

**Important:** The temp file lifecycle requires moving the cleanup responsibility to the caller. The
`BuildDockerRunCommand` function returns the temp file path alongside the args so the caller can
clean up after `docker run` completes. Alternatively, the function creates and cleans up the file
internally, returning args that reference the file — but the file must exist when `docker run`
executes, so cleanup happens in the provider's `Create` method after `RunContainer` returns.

#### GCP Backend

Custom env vars are added to the instance metadata map with a `SPINNER_ENV_` prefix:

```go
for key, value := range config.EnvVars {
    metadata["SPINNER_ENV_"+key] = value
}
```

The `gcp_runtime.sh` script reads all `SPINNER_ENV_*` metadata keys and exports them (stripping the
prefix):

```bash
# Read custom env vars from metadata (SPINNER_ENV_ prefix)
for key in $(curl -sf -H "$META_HEADER" "$META_URL/" | grep "^SPINNER_ENV_"); do
    value=$(curl -sf -H "$META_HEADER" "$META_URL/$key" || echo "")
    export "${key#SPINNER_ENV_}=$value"
done
```

The prefix avoids collisions with Spinner's internal metadata keys (REPO_URL, PROMPT, etc.).

#### Validation

The `--env` flag values are validated before creating the provider:

1. Each value must contain `=` (e.g., `KEY=VALUE`)
2. The key portion (before first `=`) must be non-empty
3. The key must not conflict with Spinner's reserved env vars: `GITHUB_TOKEN`,
   `CLAUDE_CODE_OAUTH_TOKEN`, `REPO_URL`, `PROMPT`, `BRANCH`, `MAX_ITERATIONS`,
   `LOG_DIR`, `STATE_DIR`, `SPINNER_LOG_BUCKET`, `SPINNER_STATE_BUCKET`, `SPINNER_INSTANCE_NAME`

Reserved var conflicts produce a clear error: `--env: cannot override reserved variable "GITHUB_TOKEN"`.

### Key Decisions

| Decision | Rationale |
|---|---|
| `--env-file` for Docker (not bare `-e`) | Keeps secrets out of `ps aux`; negligible complexity increase |
| `map[string]string` in `CreateConfig` | Deduplication, clean iteration, both backends consume identically |
| `SPINNER_ENV_` prefix for GCP metadata | Avoids collision with internal metadata keys; easy to enumerate |
| Validate against reserved vars | Prevents silent override of auth tokens or internal config |
| General flag (not backend-specific) | Works identically on Docker and GCP; no conditional validation |
| No `--env-file` CLI flag (file input) | Keep it simple; users can use shell: `--env "$(cat .env)"`. Add later if needed |
