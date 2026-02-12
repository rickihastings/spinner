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
| `internal/secret/resolver.go` | **create** | `Resolve(store, customKeys)` — store → env → error |
| `internal/secret/resolver_test.go` | **create** | Resolution order and error condition tests |
| `internal/secret/mock_store.go` | **create** | Testify MockStore for consumer tests |
| `cmd/secret.go` | **create** | `spinner secret` subcommand (set/list/delete) |
| `cmd/secret_test.go` | **create** | Subcommand tests with MockStore injection |
| `cmd/helpers.go` | **modify** | Add `flagSecret` constant |
| `cmd/spin.go` | **modify** | Add `--secret` flag, create Store, resolve secrets, populate Secrets |
| `cmd/spin_test.go` | **modify** | Test `--secret` flag parsing and validation |
| `internal/provider/provider.go` | **modify** | Add `Secrets map[string]string` to `CreateConfig` |
| `internal/backend/docker/run.go` | **modify** | Read tokens from `config.Secrets` instead of `os.Getenv()` |
| `internal/backend/docker/run_test.go` | **modify** | Pass `Secrets` in `spinConfig` |
| `internal/backend/docker/docker_provider.go` | **modify** | Map `CreateConfig.Secrets` → `spinConfig.Secrets` |
| `internal/backend/gcp/gcp_provider.go` | **modify** | Read tokens from `config.Secrets` instead of `os.Getenv()` |
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

#### Container Delivery (Unchanged)

- **Docker:** Secrets from `config.Secrets` are written to the temp env file alongside other vars
  (same `0600` permissions, same `defer os.Remove()` cleanup)
- **GCP:** Secrets from `config.Secrets` are written to instance metadata

No changes to the container delivery mechanism. The improvement is entirely on the host side.

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

### Dependencies

- `golang.org/x/crypto v0.45.0` (already indirect in go.mod) — for `argon2` package
- `golang.org/x/term v0.37.0` (already indirect in go.mod) — for `ReadPassword` (hidden passphrase input)

No new external modules required. Both are promoted from indirect to direct usage.
