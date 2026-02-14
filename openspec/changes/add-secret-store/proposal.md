# Proposal: Add Secret Store

## Summary

Add an encrypted secret store (`internal/secret/`) backed by an AES-256-GCM encrypted file, a
`spinner secret set/list/delete` CLI subcommand, and a `--secret KEY` flag on `spin` that references keys
from the store. Built-in tokens (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`) auto-resolve from the store
first, then fall back to environment variables for backward compatibility.

## Motivation

Spinner reads `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from host environment variables and forwards
them into containers — the host CLI never uses these tokens itself, it just passes them through. In
practice, users store these in `.envrc` files on disk — plaintext secrets visible to any process with
filesystem access. This is particularly concerning when running agents autonomously:

- **Plaintext on disk** — `.envrc` files contain sensitive tokens readable by any local process (including
  the autonomous agents Spinner orchestrates)
- **Shell history exposure** — `--env KEY=VALUE` puts secret values in shell history and `ps aux`
- **No separation** — non-sensitive config and sensitive tokens share the same `.envrc` file with no
  distinction between them
- **No secure storage** — there is no way to store secrets outside plaintext files today

An encrypted file store gives users a single, cross-platform way to keep tokens out of plaintext files.
At spin-time, the store is unlocked once, values are read and forwarded into the container — the same
pass-through model as today, but with encrypted storage instead of `.envrc`.

## What Changes

### Added Capability: Secret Store

New `internal/secret/` package with:

- **Store interface** — `Set`, `Get`, `Delete`, `List` methods with `ErrNotFound` sentinel
- **Encrypted file backend** — AES-256-GCM with Argon2id key derivation (`~/.spinner/secrets.enc`)
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

### Modified Capability: In-Container Secret Delivery

Custom `--secret` values are NOT passed as container environment variables. Instead:

- **ADDED** encrypted blob delivery — host re-encrypts resolved custom secrets into a per-session blob,
  mounted read-only into the container at `/run/spinner/secrets.enc`
- **MODIFIED** `spinner exec` — reads encrypted blob at startup, decrypts into memory, deletes blob,
  injects secrets via `cmd.Env` on child processes. Secrets never appear in the container's global
  environment or filesystem after startup.
- **ADDED** `spinner secret inject -- <command>` — for no-`--prompt` mode (user SSH). Prompts for
  passphrase, decrypts blob, runs command with secrets injected. Unattended agents without explicit
  injection get no custom secrets.

Built-in tokens (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`) remain as container env vars because
`startup.sh` requires them for `gh auth setup-git` and git credential configuration.

## Impact

### Affected Specs

- `cli-spin` — new `--secret` flag, modified token resolution, new `spinner secret` subcommand,
  encrypted blob delivery, `spinner exec` secret injection, `spinner secret inject` command

### Affected Code

| Area | Change Type |
|---|---|
| `internal/secret/store.go` | New — Store interface, ErrNotFound sentinel |
| `internal/secret/encrypted.go` | New — AES-256-GCM encrypted file backend |
| `internal/secret/blob.go` | New — `EncryptBlob` / `DecryptBlob` for per-session secret transport |
| `internal/secret/resolver.go` | New — token resolution (store → env → error) |
| `internal/secret/mock_store.go` | New — testify mock for consumer tests |
| `cmd/secret.go` | New — `spinner secret` subcommand (set/list/delete/inject) |
| `cmd/helpers.go` | Modify — add `flagSecret` constant |
| `cmd/spin.go` | Modify — add `--secret` flag, create Store, resolve secrets, populate `CreateConfig.Secrets` |
| `internal/provider/provider.go` | Modify — add `Secrets` field to `CreateConfig` |
| `internal/backend/docker/run.go` | Modify — read built-in tokens from `config.Secrets`; write encrypted blob to host state dir; mount blob into container |
| `internal/backend/docker/docker_provider.go` | Modify — pass `Secrets` through to `spinConfig` |
| `internal/backend/gcp/gcp_provider.go` | Modify — read built-in tokens from `config.Secrets`; base64-encode blob as `SPINNER_SECRET_BLOB` instance metadata |
| `internal/exec/loop.go` | Modify — read and decrypt secrets blob at startup, inject into executor config |
| `internal/agent/claude/executor.go` | No change — already injects `config.Env` via `cmd.Env` |
| `internal/prerequisites/prerequisites.go` | Modify — remove `CheckEnvironmentVariables()` (replaced by resolver) |
| Test files | New and updated tests for all changes |

### Not Affected

- `--env` flag — continues to work identically for inline key-value pairs
- `--env-file` proposal — remains a separate feature for non-sensitive bulk config
- Provider interface — no new methods, just a field on `CreateConfig`
- Docker/GCP images — no changes to baked images
- `setup`, `watch` commands — no changes
- Built-in token delivery — `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` remain as container env vars
  (`startup.sh` dependency)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Encrypted file passphrase UX | Interactive prompt blocks CI | `SPINNER_SECRET_PASSPHRASE` env var for non-interactive use; CI can also use env var fallback (existing workflow) |
| Backward compatibility | Users relying on env vars must continue to work | Resolver falls back to `os.Getenv()` for built-in tokens; env-only workflow unchanged |
| Encrypted file corruption | Lost secrets | Clear error message; user can delete file and re-set secrets |
| Go memory safety | Secret strings cannot be reliably zeroed | Accepted Go limitation; all Go CLI tools share this constraint |
| Passphrase reuse | Same passphrase encrypts host store and container blob | Container blob uses its own salt; host store file is never inside the container |
| `SPINNER_SECRET_PASSPHRASE` in --prompt mode | Passphrase briefly in container env | `spinner exec` unsets it immediately after decrypting; window is sub-second |
| No-prompt mode without passphrase | User must type passphrase for every `inject` | Acceptable UX tradeoff for preventing unattended agent access |

### Added Capability: Configurable Store Path

- **ADDED** `SPINNER_SECRET_STORE` environment variable — overrides default store path
  (`~/.spinner/secrets.enc`). Enables inception scenarios where an outer Spinner's encrypted blob
  serves as the inner Spinner's store. Same passphrase, same format, each layer reads what it needs
  and re-encrypts for the next.

## Non-Goals

- **macOS Keychain / OS-native backends** — encrypted file is cross-platform and sufficient; no need for platform-specific code paths
- **GCP Secret Manager integration** — future hardening option for GCP-hosted instances
- **Docker Swarm secrets** — requires Swarm mode, incompatible with standalone containers
- **Encrypted env-file support** — `--env-file` is for non-sensitive config; sensitive values go through `--secret`
- **Key rotation automation** — users rotate by re-running `spinner secret set`
- **Multi-user / team secret sharing** — single-user tool; teams use their own secret management
