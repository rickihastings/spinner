# Proposal: Add Custom Environment Variables / Secrets

## Summary

Add a repeatable `--env KEY=VALUE` flag to the `spin` command so users can inject arbitrary environment
variables (API keys, tokens, config values) into sandboxed instances at runtime. Docker backend uses
`--env-file` with a temporary file (keeps secrets out of `ps` output). GCP backend uses instance metadata
with a `SPINNER_ENV_` prefix.

## Motivation

Currently Spinner only passes two hardcoded secrets (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`) into
sandboxes. Users need to pass additional secrets for their workflows:

- **NPM tokens** for private registries
- **API keys** for external services (databases, SaaS tools, CI systems)
- **Custom config** for project-specific tooling

The only workaround today is hardcoding secrets in a bake script (`--bake-script`), which embeds them
in the disk image — visible to anyone with image access and persisted indefinitely. Runtime injection
is the correct approach: secrets arrive when the instance starts and never touch the image.

## What Changes

### Modified Capability: `cli-spin`

- **ADDED** `--env KEY=VALUE` repeatable flag — accepts arbitrary environment variables
- **ADDED** Validation: `KEY=VALUE` format required, key must be non-empty, reserved vars rejected
- **ADDED** Reserved variable list: `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `REPO_URL`, `PROMPT`,
  `BRANCH`, `MAX_ITERATIONS`, `LOG_DIR`, `STATE_DIR`, `SPINNER_LOG_BUCKET`, `SPINNER_STATE_BUCKET`,
  `SPINNER_INSTANCE_NAME`
- **MODIFIED** Docker backend: switches from individual `-e` flags to `--env-file` with temp file
  (security improvement — secrets no longer visible in `ps aux`)
- **MODIFIED** GCP backend: custom vars added to instance metadata with `SPINNER_ENV_` prefix
- **MODIFIED** GCP runtime script: reads `SPINNER_ENV_*` metadata keys and exports them

### Modified Internal Type: `provider.CreateConfig`

- **ADDED** `EnvVars map[string]string` field for custom environment variables

## Impact

### Affected Specs

- `cli-spin` — three new requirements added (Custom Environment Variable Injection, Docker Env-File
  Secret Passing, GCP Metadata Secret Passing)

### Affected Code

| Area | Change Type |
|---|---|
| `cmd/helpers.go` | Add `flagEnv` constant |
| `cmd/spin.go` | Add `--env` flag, parse and validate, pass to `CreateConfig` |
| `internal/provider/provider.go` | Add `EnvVars` field to `CreateConfig` |
| `internal/backend/docker/run.go` | Switch to `--env-file`, add `EnvVars` to `SpinConfig` |
| `internal/backend/docker/docker_provider.go` | Pass `EnvVars` through, handle temp file cleanup |
| `internal/backend/gcp/gcp_provider.go` | Add `SPINNER_ENV_` prefixed metadata |
| `templates/scripts/gcp_runtime.sh` | Read and export `SPINNER_ENV_*` keys |
| Test files | New and updated tests for all changes |

### Not Affected

- `setup` command — env vars are runtime-only, not bake-time
- `watch` command — no env var injection needed
- `exec` command — runs inside the container, env vars are already set
- Provider interface — no new methods, just a field on `CreateConfig`
- Docker image / GCP image — no changes to baked images

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Temp file persists on crash | Secret on disk | `defer os.Remove()` + file permissions `0600`; short-lived (deleted after `docker run`) |
| GCP metadata visible in Console | Secret in GCP UI | Same threat model as existing tokens; document that Secret Manager is a future hardening option |
| Reserved var list becomes stale | New internal vars could collide | Centralize list as a constant; update when adding internal vars |

## Non-Goals

- **`--env-file` CLI flag** (reading vars from a user-provided file) — can be added later; users can
  use shell expansion in the meantime: `--env "$(cat .secrets | head -1)"`
- **GCP Secret Manager integration** — future hardening option; metadata approach is sufficient for
  single-tenant VMs
- **Bind-mounted secret files for Docker** — better `docker inspect` protection but requires entrypoint
  changes; not justified for a developer CLI tool (see `design.md` for full analysis)
- **Env vars for `setup` command** — bake-time secrets should use BuildKit `--secret` if needed; that's
  a separate concern
