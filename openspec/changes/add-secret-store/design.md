# Design: Add Secret Store

## Key Insight: Tokens Are Pass-Through

Spinner never uses `GITHUB_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` on the host. It reads them from
environment variables and forwards them into containers:

- `internal/backend/docker/run.go:120-121` — writes to temp env file
- `internal/backend/gcp/gcp_provider.go:185-186` — writes to instance metadata
- `internal/prerequisites/prerequisites.go:20-33` — validates non-empty

The host CLI is a pass-through. This means the secret store only needs to provide values at
spin-time — unlock once, read values, forward into the container. No persistent runtime access,
no background processes, no daemon.

## Why Encrypted File Only

We evaluated multiple secret storage approaches:

| Option | Pros | Cons |
|---|---|---|
| macOS Keychain via `security` CLI | Native, hardware-backed | macOS only, requires CGo or CLI exec, platform-specific codepath |
| OS-native per-platform | Fully native | 3x implementation, testing complexity; GNOME Keyring needs D-Bus |
| External tools (`pass`, `vault`) | Rich features | External dependency users must install and configure |
| **Encrypted file (AES-256-GCM + Argon2id)** | **Cross-platform, pure Go, single codepath** | **Requires passphrase** |

**Decision: Encrypted file only.**

Rationale:
- **One codepath everywhere** — same implementation on macOS, Linux, CI. No platform detection,
  no conditional backends, half the code.
- **Pure Go** — no CGo, no external binaries, no `exec.Command` to mock. `golang.org/x/crypto`
  (already in go.mod) provides everything.
- **Passphrase UX is manageable** — `SPINNER_SECRET_PASSPHRASE` env var for CI/scripts, interactive
  prompt for local development. And for users who don't want any of this, env vars still work
  (backward compat).
- **Tokens are pass-through** — we don't need the sophistication of Keychain for a tool that just
  reads secrets once at startup and forwards them. An encrypted file is sufficient.

---

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|---|---|---|
| `internal/secret/store.go` | **create** | Store interface, ErrNotFound sentinel error |
| `internal/secret/encrypted.go` | **create** | EncryptedFileStore — AES-256-GCM + Argon2id |
| `internal/secret/encrypted_test.go` | **create** | Round-trip, corruption, wrong-passphrase tests |
| `internal/secret/blob.go` | **create** | `EncryptBlob` / `DecryptBlob` — per-session secret transport |
| `internal/secret/blob_test.go` | **create** | Round-trip, wrong passphrase, corrupted blob tests |
| `internal/secret/resolver.go` | **create** | `Resolve(store, customKeys)` — store → env → error |
| `internal/secret/resolver_test.go` | **create** | Resolution order and error condition tests |
| `internal/secret/mock_store.go` | **create** | Testify MockStore for consumer tests |
| `cmd/secret.go` | **create** | `spinner secret` subcommand (set/list/delete/inject) |
| `cmd/secret_test.go` | **create** | Subcommand tests with MockStore injection |
| `cmd/helpers.go` | **modify** | Add `flagSecret` constant |
| `cmd/spin.go` | **modify** | Add `--secret` flag, create Store, resolve secrets, populate Secrets |
| `cmd/spin_test.go` | **modify** | Test `--secret` flag parsing and validation |
| `internal/provider/provider.go` | **modify** | Add `Secrets map[string]string` to `CreateConfig` |
| `internal/backend/docker/run.go` | **modify** | Read tokens from `config.Secrets`; write blob to host state dir; mount blob into container |
| `internal/backend/docker/run_test.go` | **modify** | Pass `Secrets` in `spinConfig` |
| `internal/backend/docker/docker_provider.go` | **modify** | Map `CreateConfig.Secrets` → `spinConfig.Secrets` |
| `internal/backend/gcp/gcp_provider.go` | **modify** | Read tokens from `config.Secrets`; base64-encode blob as `SPINNER_SECRET_BLOB` instance metadata |
| `internal/exec/loop.go` | **modify** | Decrypt secrets blob at startup, inject into executor config |
| `internal/prerequisites/prerequisites.go` | **modify** | Remove `CheckEnvironmentVariables()` (replaced by resolver) |
| `docs/usage.md` | **modify** | Document `spinner secret` workflow and `--secret` flag |

### Approach

#### Store Interface

```go
package secret

import "errors"

var ErrNotFound = errors.New("secret not found")

type Store interface {
    Set(key, value string) error
    Get(key string) (string, error)   // returns ErrNotFound if key does not exist
    Delete(key string) error
    List() ([]string, error)          // returns key names only
}
```

Small focused interface following the project's pattern (`Provider`, `DockerClient`). The
`ErrNotFound` sentinel lets callers distinguish "not found" from backend failures.

#### Encrypted File Store

```go
type EncryptedFileStore struct {
    path       string                  // default: ~/.spinner/secrets.enc
    passphrase func() (string, error)  // prompt or SPINNER_SECRET_PASSPHRASE env var
}
```

- **File format:** 16-byte Argon2id salt + 12-byte AES-GCM nonce + ciphertext
- **Plaintext format:** JSON-encoded `map[string]string`
- **Key derivation:** Argon2id with `time=1, memory=64*1024, threads=4`
- **Passphrase source:** `SPINNER_SECRET_PASSPHRASE` env var first, then interactive prompt via
  `golang.org/x/term.ReadPassword()`
- **Atomic writes:** Write to temp file, then `os.Rename()` to prevent corruption on crash
- **File permissions:** `0600` (owner read/write only)
- **Missing file:** Treated as empty store (first `Set` creates the file)
- **Operations:** All mutating operations load-decrypt-modify-encrypt-write atomically

The store is unlocked once per CLI invocation (the passphrase function is called on first
access). Since tokens are pass-through, there's only one unlock per `spin` command.

#### Secret Resolver

```go
func Resolve(store Store, customKeys []string) (map[string]string, error)
```

Returns a single `map[string]string` containing all resolved secrets (built-in tokens + custom keys).

Resolution order per key:
1. `store.Get(key)` — if found, use it
2. `os.Getenv(key)` — for built-in tokens only (backward compat)
3. For built-in tokens: error if neither source has a value
4. For custom `--secret` keys: must exist in store (no env fallback)

The resolver replaces the three scattered `os.Getenv` call sites:
- `prerequisites.CheckEnvironmentVariables()` — removed, resolver subsumes this
- `docker/run.go:120-121` — reads from `config.Secrets` instead
- `gcp_provider.go:185-186` — reads from `config.Secrets` instead

#### Spin Command Integration

Resolved values are placed in `CreateConfig.Secrets` so backends receive pre-resolved values
via config rather than calling `os.Getenv()` themselves. This matches the existing `EnvVars`
pattern and keeps backends ignorant of the secret storage mechanism.

```go
// In cmd/spin.go RunE:
store := secret.NewEncryptedFileStore(defaultPath, passphraseFunc)
resolved, err := secret.Resolve(store, spinSecrets)  // spinSecrets from --secret flags
// ...
createConfig := provider.CreateConfig{
    // ...existing fields...
    Secrets: resolved,
}
```

#### Container Delivery (Encrypted Blob)

Custom `--secret` values are NOT passed as container environment variables. They are delivered as an
encrypted blob that requires explicit decryption inside the container.

**Split: Built-in Tokens vs Custom Secrets**

| Token Type | Delivery | Reason |
|---|---|---|
| `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN` | Container env vars (unchanged) | `startup.sh` requires them for `gh auth setup-git` and git credential config |
| Custom `--secret` values | Encrypted blob at `/run/spinner/secrets.enc` | Never exposed as container env vars |

**Blob Generation (host side, at spin-time):**

```go
// In cmd/spin.go RunE, after Resolve():
// builtinSecrets go to CreateConfig.Secrets (env var delivery, unchanged)
// customSecrets go to encrypted blob
customSecrets := filterCustomSecrets(resolved, builtinKeys)
if len(customSecrets) > 0 {
    blob, err := secret.EncryptBlob(customSecrets, passphrase)
    createConfig.SecretBlob = blob  // backends mount/upload this
}
```

The blob uses the same AES-256-GCM + Argon2id scheme as the host store but with a fresh salt.
The user's store passphrase encrypts the blob — same passphrase, different salt, separate file.

**Docker Backend:**

1. Host writes encrypted blob to `~/.spinner/<container>/secrets.enc` (alongside existing state dir)
2. Mounted read-only into container at `/run/spinner/secrets.enc` via `-v` flag
3. For `--prompt` mode: `SPINNER_SECRET_PASSPHRASE` passed as container env var
4. For no-`--prompt` mode: passphrase NOT in container env

**GCP Backend:**

1. Host base64-encodes the encrypted blob and passes it as instance metadata key `SPINNER_SECRET_BLOB`
2. Startup script decodes the metadata value and writes it to `/run/spinner/secrets.enc`
3. For `--prompt` mode: passphrase passed as instance metadata key `SPINNER_SECRET_PASSPHRASE`
4. For no-`--prompt` mode: passphrase NOT in metadata
5. Blob is destroyed when the VM is deleted — no orphaned files in GCS

#### `--prompt` Mode: `spinner exec` as Secret Broker

When `spinner exec` starts inside the container:

```go
// In internal/exec/loop.go, before entering iteration loop:
blobPath := "/run/spinner/secrets.enc"
passphrase := os.Getenv("SPINNER_SECRET_PASSPHRASE")
if passphrase != "" && fileExists(blobPath) {
    secrets, err := secret.DecryptBlob(blobPath, passphrase)
    os.Remove(blobPath)                          // delete blob from filesystem
    os.Unsetenv("SPINNER_SECRET_PASSPHRASE")     // remove passphrase from own env
    // secrets held in memory, injected into executor config
    executorConfig.Env = append(executorConfig.Env, secretsToEnvSlice(secrets)...)
}
```

1. Read blob from `/run/spinner/secrets.enc`
2. Read passphrase from `SPINNER_SECRET_PASSPHRASE` env var
3. Decrypt secrets into memory
4. Delete the blob file from disk
5. Unset `SPINNER_SECRET_PASSPHRASE` from own process environment
6. Inject secrets via `cmd.Env` when spawning Claude CLI (existing mechanism at `executor.go:83-85`)

After startup, secrets exist only in `spinner exec`'s heap memory. They are not on the filesystem,
not in the container's global environment, and not discoverable via `docker exec env` or
`/proc/1/environ`. They ARE in each Claude CLI child process's `/proc/<pid>/environ` while it runs,
which is acceptable — the agent needs them to do work.

#### No-`--prompt` Mode: `spinner secret inject`

When user SSHs into the container:

1. Blob exists at `/run/spinner/secrets.enc` (encrypted, unreadable without passphrase)
2. `SPINNER_SECRET_PASSPHRASE` is NOT in the container environment
3. User runs `spinner secret inject -- <command>` to access secrets:

```bash
# Decrypt and inject for a specific command
spinner secret inject -- claude -p "implement feature X"

# Or start a subshell with secrets available
spinner secret inject -- bash
```

`spinner secret inject` implementation:

```go
// In cmd/secret.go:
func runSecretInject(cmd *cobra.Command, args []string) error {
    passphrase := promptPassphrase()                    // hidden input via x/term
    secrets, err := secret.DecryptBlob(blobPath, passphrase)
    // Run the command with secrets injected
    child := exec.Command(args[0], args[1:]...)
    child.Env = append(os.Environ(), secretsToEnvSlice(secrets)...)
    child.Stdin = os.Stdin
    child.Stdout = os.Stdout
    child.Stderr = os.Stderr
    return child.Run()
}
```

**Key security property:** An unattended agent started via `docker exec` or in a separate SSH session
does NOT have access to custom secrets. It would need the passphrase to decrypt the blob, and the
passphrase is never in the container environment in no-`--prompt` mode. The agent sees `GITHUB_TOKEN`
(needed for git operations) but not `NPM_TOKEN`, `API_KEY`, or any other custom secrets.

### Key Decisions

| Decision | Rationale |
|---|---|
| Encrypted file only (no Keychain) | Single codepath, cross-platform, pure Go. Tokens are pass-through so Keychain sophistication isn't needed |
| Secrets in `CreateConfig` (not Store in backends) | Backends stay ignorant of secret storage. Resolution happens once at command level. Matches existing `EnvVars` pattern |
| `--secret` separate from `--env` | `--env KEY=VALUE` exposes value on CLI. `--secret KEY` references store only. Different security semantics |
| Argon2id + AES-256-GCM | Current best practice for password-based key derivation + authenticated encryption |
| No env fallback for custom `--secret` keys | `--secret NPM_TOKEN` must exist in store. Silent env fallback undermines security intent. Built-in tokens get env fallback for backward compat only |
| `~/.spinner/secrets.enc` location | Consistent with existing `~/.spinner/` config directory |
| `SPINNER_SECRET_PASSPHRASE` env var | Standard CI escape hatch; alternatively CI users set tokens as env vars directly (existing workflow) |
| Encrypted blob (not env vars) for custom secrets | Prevents unattended agents from reading custom secrets via `env`. Built-in tokens remain env vars because `startup.sh` requires them |
| Same passphrase for host store and container blob | Single passphrase UX. Container blob has its own salt. Host store file never enters the container |
| Passphrase in env only for --prompt mode | `spinner exec` needs non-interactive decryption. Unsets immediately. No-prompt mode requires interactive passphrase to gate agent access |
| `spinner secret inject` wrapper (not global export) | Limits secret exposure to explicit command trees. User controls which processes get secrets |
| `spinner exec` deletes blob + unsets passphrase | Minimizes window of exposure. After startup, secrets exist only in process memory |

#### Inception: Spinner Inside Spinner

A common pattern is: local machine → GCP VM → Docker containers inside the VM. Each layer needs
access to secrets without the user's host store being available.

The encrypted blob solves this because **blob format = store format**. The store path is configurable
via `SPINNER_SECRET_STORE` environment variable (default: `~/.spinner/secrets.enc`). At each layer:

```
Layer 0 (local machine):
  spinner spin --backend gcp --secret NPM_TOKEN --secret API_KEY ...
  → reads from ~/.spinner/secrets.enc (host store)
  → generates encrypted blob → passes as instance metadata
  → VM gets /run/spinner/secrets.enc

Layer 1 (GCP VM, user SSHs in):
  SPINNER_SECRET_STORE=/run/spinner/secrets.enc \
    spinner spin --backend docker --secret NPM_TOKEN --repo ...
  → reads NPM_TOKEN from /run/spinner/secrets.enc (outer blob, same format as store)
  → generates new blob → mounts into Docker container at /run/spinner/secrets.enc

Layer 2 (Docker container, --prompt mode):
  spinner exec reads blob → decrypts → injects into Claude CLI
```

Same passphrase at every layer. The encrypted file travels downward, each layer decrypting what it
needs and re-encrypting for the next. No store setup required inside VMs or containers — the blob
IS the store.

**Implementation:** `EncryptedFileStore` already takes a configurable `path`. The only addition is
checking `SPINNER_SECRET_STORE` env var before defaulting to `~/.spinner/secrets.enc`:

```go
func defaultStorePath() string {
    if p := os.Getenv("SPINNER_SECRET_STORE"); p != "" {
        return p
    }
    return filepath.Join(home, ".spinner", "secrets.enc")
}
```

### Dependencies

- `golang.org/x/crypto v0.45.0` (already indirect in go.mod) — for `argon2` package
- `golang.org/x/term v0.37.0` (already indirect in go.mod) — for `ReadPassword` (hidden passphrase input)

No new external modules required. Both are promoted from indirect to direct usage.
