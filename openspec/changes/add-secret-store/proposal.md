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
- **ADDED** `SecretKey []byte` field — carries the ephemeral 32-byte random key for blob decryption
- **REMOVED** direct token fields — backends no longer read `GITHUB_TOKEN` from env or config; all
  secrets travel via the encrypted blob
- **REMOVED** `Passphrase string` field — replaced by `SecretKey`

### Modified Capability: In-Container Secret Delivery

**No secrets or decryption keys are passed as environment variables or instance metadata.** All
secrets are delivered as an encrypted blob. The decryption key is delivered via a file-based side
channel separate from the blob:

- **ADDED** ephemeral key transport — at spin-time, a random 32-byte AES-256-GCM key encrypts the
  blob directly (no Argon2id — unnecessary for random keys). The key is delivered via file, never
  as an env var or instance metadata value.
- **ADDED** encrypted blob delivery — host encrypts all resolved secrets into a per-session blob,
  mounted read-only into the container at `/run/spinner/secrets.enc`
- **MODIFIED** Docker delivery — ephemeral key written to `~/.spinner/<container>/secrets.key`,
  mounted read-only at `/run/spinner/secrets.key`. No passphrase in env-file or `docker inspect`.
- **MODIFIED** GCP delivery — ephemeral key written to GCS at
  `gs://<state-bucket>/<instance>/secrets.key`. Startup script fetches key from GCS. Key never
  appears in instance metadata. Key persists in GCS for VM restart support (re-read on each boot).
- **MODIFIED** `startup.sh` — uses `spinner secret inject` to wrap token-dependent operations
  (`gh auth setup-git`, `git clone`). Reads key from `/run/spinner/secrets.key`. After initial
  setup, `gh` credential cache persists git auth and the token env vars are no longer needed.
- **MODIFIED** `spinner exec` — reads encrypted blob and key file at startup, decrypts into memory,
  injects secrets via `cmd.Env` on child processes. Key file and blob are retained on disk for
  inception scenarios.
- **ADDED** `spinner secret inject -- <command>` — for no-`--prompt` mode (user SSH). Reads key
  from file (or prompts for passphrase as fallback), decrypts blob, runs command with secrets
  injected. Unattended agents without explicit injection get no secrets at all.

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
| `internal/backend/docker/run.go` | Modify — mount encrypted blob + key file; remove env-file token writing; no passphrase in env |
| `internal/backend/docker/docker_provider.go` | Modify — pass `SecretBlob` + `SecretKey` from `CreateConfig` to `spinConfig` |
| `internal/backend/gcp/gcp_provider.go` | Modify — base64-encode blob as `SPINNER_SECRET_BLOB` metadata; upload key to GCS (not metadata) |
| `internal/backend/gcp/templates/scripts/gcp_runtime.sh` | Modify — fetch key from GCS instead of metadata; write to `/run/spinner/secrets.key` |
| `internal/exec/loop.go` | Modify — decrypt blob using key file at startup, inject into executor config |
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
| Encrypted file passphrase UX | Interactive prompt blocks CI | `SPINNER_SECRET_PASSPHRASE` env var for non-interactive use on the host (host only — never sent to containers) |
| Encrypted file corruption | Lost secrets | Clear error message; user can delete file and re-set secrets |
| Go memory safety | Secret strings cannot be reliably zeroed | Accepted Go limitation; all Go CLI tools share this constraint |
| Key file on same Docker mount as blob | Key and blob co-located on host volume | Security boundary is Docker isolation; key-as-file eliminates `docker inspect` leak of passphrase |
| Key in GCS bucket | Accessible to anyone with `storage.objects.get` on bucket | Separate IAM from `compute.instances.get`; key not visible in GCP Console VM details; bucket already exists for logs/state |
| Blob retained on disk | Encrypted file persists in container filesystem | Encrypted with random AES key; useless without key file; enables inception |

## Non-Goals

- **macOS Keychain / OS-native backends** — encrypted file is cross-platform and sufficient; no need for platform-specific code paths
- **Docker Swarm secrets** — requires Swarm mode, incompatible with standalone containers
- **Encrypted env-file support** — `--env-file` is for non-sensitive config; sensitive values go through `--secret`
- **Key rotation automation** — users rotate by re-running `spinner secret set`
- **Multi-user / team secret sharing** — single-user tool; teams use their own secret management
- **Environment variable fallback** — deliberately removed to eliminate plaintext token workflows
