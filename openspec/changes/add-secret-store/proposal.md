# Proposal: Add Secret Store

## Summary

Add an encrypted secret store (`internal/secret/`) backed by an AES-256-GCM encrypted file, a
`spinner secret set/list/delete` CLI subcommand, and a `--secret KEY` flag on `spin` that references keys
from the store. **All tokens are stored in the secret store — there is no environment variable fallback.**
This is a breaking change from the current env-var-based workflow.

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
- **Env vars leak into containers** — tokens passed as container env vars are visible to every process
  inside the sandbox, including unattended agents the user may spin up

An encrypted file store gives users a single, cross-platform way to keep tokens out of plaintext files.
At spin-time, the store is unlocked once, all secrets are encrypted into a per-session blob and delivered
into the container. Inside the container, secrets are never exposed as environment variables — they are
decrypted on demand by `spinner exec` or `spinner secret inject`.

## What Changes

### Breaking Change: Store-Only Token Resolution

`GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` **must** be stored in the secret store via
`spinner secret set`. Environment variable fallback is removed. Users must run:

```bash
spinner secret set GITHUB_TOKEN
spinner secret set CLAUDE_CODE_OAUTH_TOKEN
```

before `spinner spin` will work. This eliminates plaintext `.envrc` files from the workflow entirely.

### Added Capability: Secret Store

New `internal/secret/` package with:

- **Store interface** — `Set`, `Get`, `Delete`, `List` methods with `ErrNotFound` sentinel
- **Encrypted file backend** — AES-256-GCM with Argon2id key derivation (`~/.spinner/secrets.enc`)
- **Resolver** — centralizes token resolution: store only, no env fallback

### Added Capability: `spinner secret` CLI Subcommand

- `spinner secret set <KEY>` — prompts for value (hidden input) or accepts `--value`
- `spinner secret list` — shows stored key names (not values)
- `spinner secret delete <KEY>` — removes a key from the store
- `spinner secret inject -- <command>` — decrypts blob, runs command with secrets injected

### Modified Capability: `cli-spin`

- **ADDED** `--secret KEY` repeatable flag — references a key in the secret store by name (value never on
  command line)
- **MODIFIED** token resolution — all tokens (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, custom keys)
  resolve from the secret store only. No env var fallback. Error if key not found in store.
- **REMOVED** reserved variable check for `--secret` — `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` are
  no longer special-cased; they are regular store keys resolved automatically by the resolver

### Modified Internal Type: `provider.CreateConfig`

- **ADDED** `SecretBlob []byte` field — carries the encrypted blob for backend mounting/delivery
- **REMOVED** direct token fields — backends no longer read `GITHUB_TOKEN` from env or config; all
  secrets travel via the encrypted blob

### Modified Capability: In-Container Secret Delivery

**No secrets are passed as container environment variables.** All secrets (built-in tokens and custom)
are delivered as an encrypted blob:

- **ADDED** encrypted blob delivery — host encrypts all resolved secrets into a per-session blob,
  mounted read-only into the container at `/run/spinner/secrets.enc`
- **MODIFIED** `startup.sh` — uses `spinner secret inject` to wrap token-dependent operations
  (`gh auth setup-git`, `git clone`). After initial setup, `gh` credential cache persists git auth
  and the token env vars are no longer needed.
- **MODIFIED** `spinner exec` — reads encrypted blob at startup, decrypts into memory, injects secrets
  (including `SPINNER_SECRET_PASSPHRASE` for inception) via `cmd.Env` on child processes. Blob is
  retained on disk for inception scenarios.
- **ADDED** `spinner secret inject -- <command>` — for no-`--prompt` mode (user SSH). Prompts for
  passphrase, decrypts blob, runs command with secrets injected. Unattended agents without explicit
  injection get no secrets at all.

### Added Capability: Configurable Store Path

- **ADDED** `SPINNER_SECRET_STORE` environment variable — overrides default store path
  (`~/.spinner/secrets.enc`). Enables inception scenarios where an outer Spinner's encrypted blob
  serves as the inner Spinner's store. Same passphrase, same format, each layer reads what it needs
  and re-encrypts for the next.

## Impact

### Affected Specs

- `cli-spin` — new `--secret` flag, store-only token resolution, new `spinner secret` subcommand,
  encrypted blob delivery, `spinner exec` secret injection, `spinner secret inject` command,
  `startup.sh` refactor

### Affected Code

| Area | Change Type |
|---|---|
| `internal/secret/store.go` | New — Store interface, ErrNotFound sentinel |
| `internal/secret/encrypted.go` | New — AES-256-GCM encrypted file backend |
| `internal/secret/blob.go` | New — `EncryptBlob` / `DecryptBlob` for per-session secret transport |
| `internal/secret/resolver.go` | New — token resolution (store only, no env fallback) |
| `internal/secret/mock_store.go` | New — testify mock for consumer tests |
| `cmd/secret.go` | New — `spinner secret` subcommand (set/list/delete/inject) |
| `cmd/helpers.go` | Modify — add `flagSecret` constant |
| `cmd/spin.go` | Modify — add `--secret` flag, create Store, resolve all secrets, generate blob |
| `internal/provider/provider.go` | Modify — add `SecretBlob []byte` to `CreateConfig`, remove direct token access |
| `internal/backend/docker/run.go` | Modify — mount encrypted blob; remove env-file token writing; pass `SPINNER_SECRET_PASSPHRASE` as sole container env var |
| `internal/backend/docker/docker_provider.go` | Modify — pass `SecretBlob` from `CreateConfig` to `spinConfig` |
| `internal/backend/gcp/gcp_provider.go` | Modify — base64-encode blob as `SPINNER_SECRET_BLOB` metadata; pass `SPINNER_SECRET_PASSPHRASE` as metadata |
| `internal/exec/loop.go` | Modify — decrypt blob at startup, inject into executor config (including passphrase for inception) |
| `internal/agent/claude/executor.go` | No change — already injects `config.Env` via `cmd.Env` |
| `internal/prerequisites/prerequisites.go` | Modify — remove `CheckEnvironmentVariables()` (replaced by resolver) |
| `templates/scripts/startup.sh` | Modify — refactor to use `spinner secret inject` for token-dependent work |
| Test files | New and updated tests for all changes |

### Not Affected

- `--env` flag — continues to work identically for inline non-sensitive key-value pairs
- `--env-file` proposal — remains a separate feature for non-sensitive bulk config
- Provider interface — no new methods
- Docker/GCP images — no changes to baked images
- `setup`, `watch` commands — no changes

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| **Breaking change** | Existing env-var workflows stop working | No users yet; clean migration path (`spinner secret set` for each token) |
| Encrypted file passphrase UX | Interactive prompt blocks CI | `SPINNER_SECRET_PASSPHRASE` env var for non-interactive use on the host |
| Encrypted file corruption | Lost secrets | Clear error message; user can delete file and re-set secrets |
| Go memory safety | Secret strings cannot be reliably zeroed | Accepted Go limitation; all Go CLI tools share this constraint |
| Passphrase reuse | Same passphrase encrypts host store and container blob | Container blob uses its own salt; host store file is never inside the container |
| `SPINNER_SECRET_PASSPHRASE` in container env | Passphrase discoverable via `/proc/1/environ` (Docker) or metadata API (GCP) | Defense in depth — secrets aren't casually visible via `env`; determined access requires knowing to look in /proc or metadata |
| Blob retained on disk | Encrypted file persists in container filesystem | Encrypted with Argon2id; useless without passphrase; enables inception |

## Non-Goals

- **macOS Keychain / OS-native backends** — encrypted file is cross-platform and sufficient; no need for platform-specific code paths
- **GCP Secret Manager integration** — future hardening option for GCP-hosted instances
- **Docker Swarm secrets** — requires Swarm mode, incompatible with standalone containers
- **Encrypted env-file support** — `--env-file` is for non-sensitive config; sensitive values go through `--secret`
- **Key rotation automation** — users rotate by re-running `spinner secret set`
- **Multi-user / team secret sharing** — single-user tool; teams use their own secret management
- **Environment variable fallback** — deliberately removed to eliminate plaintext token workflows
