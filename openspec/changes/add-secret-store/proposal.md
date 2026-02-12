# Proposal: Add Secret Store

## Summary

Add a secret store abstraction (`internal/secret/`) with macOS Keychain and encrypted file backends, a
`spinner secret set/list/delete` CLI subcommand, and a `--secret KEY` flag on `spin` that references keys
from the store. Built-in tokens (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`) auto-resolve from the store
first, then fall back to environment variables for backward compatibility.

## Motivation

Spinner currently reads `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from host environment variables via
`os.Getenv()`. In practice, users store these in `.envrc` files on disk — plaintext secrets visible to any
process with filesystem access. This is particularly concerning when running agents autonomously:

- **Plaintext on disk** — `.envrc` files contain sensitive tokens readable by any local process (including
  the autonomous agents Spinner orchestrates)
- **Shell history exposure** — `--env KEY=VALUE` puts secret values in shell history and `ps aux`
- **No separation** — non-sensitive config and sensitive tokens share the same `.envrc` file with no
  distinction between them
- **No secure storage** — there is no way to store secrets outside the filesystem today

macOS Keychain provides hardware-backed secure storage that most Spinner users already have access to. An
encrypted file backend covers Linux/CI environments. Together they give users a way to keep tokens out of
plaintext files entirely.

## What Changes

### Added Capability: Secret Store

New `internal/secret/` package with:

- **Store interface** — `Set`, `Get`, `Delete`, `List` methods with `ErrNotFound` sentinel
- **Keychain backend** — macOS `security` CLI for native Keychain integration (no CGo)
- **Encrypted file backend** — AES-256-GCM with Argon2id key derivation for Linux/CI
- **Auto-detection** — selects Keychain on macOS, encrypted file elsewhere
- **Resolver** — centralizes token resolution: store first, then env var fallback

### Added Capability: `spinner secret` CLI Subcommand

- `spinner secret set <KEY>` — prompts for value (hidden input) or accepts `--value`
- `spinner secret list` — shows stored key names (not values)
- `spinner secret delete <KEY>` — removes a key from the store

### Modified Capability: `cli-spin`

- **ADDED** `--secret KEY` repeatable flag — references a key in the secret store by name (value never on
  command line)
- **MODIFIED** token resolution — `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` resolve from secret store
  first, then `os.Getenv()` fallback; error only if neither source has a value
- **MODIFIED** reserved variable check — `--secret` rejects the same reserved names as `--env`

### Modified Internal Type: `provider.CreateConfig`

- **ADDED** `Secrets map[string]string` field — carries resolved secret values (built-in tokens + custom
  `--secret` keys) so backends read from config instead of calling `os.Getenv()` directly

## Impact

### Affected Specs

- `cli-spin` — new `--secret` flag, modified token resolution, new `spinner secret` subcommand

### Affected Code

| Area | Change Type |
|---|---|
| `internal/secret/store.go` | New — Store interface, ErrNotFound sentinel |
| `internal/secret/keychain.go` | New — macOS Keychain backend |
| `internal/secret/encrypted.go` | New — AES-256-GCM encrypted file backend |
| `internal/secret/detect.go` | New — backend auto-detection |
| `internal/secret/resolver.go` | New — token resolution (store → env → error) |
| `internal/secret/mock_store.go` | New — testify mock for consumer tests |
| `cmd/secret.go` | New — `spinner secret` subcommand (set/list/delete) |
| `cmd/helpers.go` | Modify — add `flagSecret` constant |
| `cmd/spin.go` | Modify — add `--secret` flag, create Store, resolve secrets, populate `CreateConfig.Secrets` |
| `internal/provider/provider.go` | Modify — add `Secrets` field to `CreateConfig` |
| `internal/backend/docker/run.go` | Modify — read tokens from `config.Secrets` instead of `os.Getenv()` |
| `internal/backend/docker/docker_provider.go` | Modify — pass `Secrets` through to `spinConfig` |
| `internal/backend/gcp/gcp_provider.go` | Modify — read tokens from `config.Secrets` instead of `os.Getenv()` |
| `internal/prerequisites/prerequisites.go` | Modify — remove `CheckEnvironmentVariables()` (replaced by resolver) |
| Test files | New and updated tests for all changes |

### Not Affected

- `--env` flag — continues to work identically for inline key-value pairs
- `--env-file` proposal — remains a separate feature for non-sensitive bulk config
- Container delivery mechanism — temp env file for Docker, metadata for GCP (unchanged)
- Provider interface — no new methods, just a field on `CreateConfig`
- Docker/GCP images — no changes to baked images
- `setup`, `watch`, `exec` commands — no secret injection needed

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| macOS Keychain access prompt | First-time system dialog asking terminal to access Keychain | Expected behavior; one-time approval per terminal app |
| Encrypted file passphrase UX | Interactive prompt blocks CI | `SPINNER_SECRET_PASSPHRASE` env var for non-interactive use; CI can also use env var fallback (existing workflow) |
| Backward compatibility | Users relying on env vars must continue to work | Resolver falls back to `os.Getenv()` for built-in tokens; env-only workflow unchanged |
| Encrypted file corruption | Lost secrets | Clear error message; user can delete file and re-set; Keychain unaffected |
| Go memory safety | Secret strings cannot be reliably zeroed | Accepted Go limitation; all Go CLI tools share this constraint |

## Non-Goals

- **GCP Secret Manager integration** — future hardening option for GCP-hosted instances
- **Docker Swarm secrets** — requires Swarm mode, incompatible with standalone containers
- **Encrypted env-file support** — `--env-file` is for non-sensitive config; sensitive values go through `--secret`
- **Key rotation automation** — users rotate by re-running `spinner secret set`
- **Multi-user / team secret sharing** — single-user tool; teams use their own secret management
