# Design: Add Secret Store

## Host-Side Secret Storage Options Analysis

Before choosing an implementation, we evaluated available mechanisms for storing secrets outside plaintext
files. The goal is to keep sensitive tokens (GitHub PATs, OAuth tokens, API keys) out of `.envrc` files
while maintaining a simple CLI workflow.

### Option 1: macOS Keychain via `security` CLI

Use the macOS Keychain Access system via the `/usr/bin/security` command-line tool.

| Property | Status |
|---|---|
| Secrets on filesystem | No (stored in encrypted Keychain database) |
| Cross-platform | macOS only |
| External dependencies | None (ships with macOS) |
| CGo required | No (CLI invocation via `exec.Command`) |

**Pros:** Native, hardware-backed, no extra dependencies, well-tested by Apple. Keychain items are
encrypted at rest and protected by the user's login password. The `security` CLI is stable across
macOS versions and avoids CGo complexity.
**Cons:** macOS only. First-time access may trigger a system dialog asking the user to allow terminal
access to Keychain.

### Option 2: Encrypted File (AES-256-GCM + Argon2id)

Store secrets in an encrypted JSON file at `~/.spinner/secrets.enc`. Key derived from a passphrase
using Argon2id, encrypted with AES-256-GCM authenticated encryption.

| Property | Status |
|---|---|
| Secrets on filesystem | Encrypted (not plaintext) |
| Cross-platform | Yes (pure Go) |
| External dependencies | `golang.org/x/crypto` (already indirect in go.mod) |
| CGo required | No |

**Pros:** Works everywhere. No external tools needed. Strong cryptographic primitives (Argon2id
resists GPU attacks, AES-256-GCM provides authenticated encryption). Passphrase can be supplied
via `SPINNER_SECRET_PASSPHRASE` env var for CI/non-interactive use.
**Cons:** Requires passphrase management. Interactive prompt for passphrase on each access (unless
env var is set). File on disk (though encrypted).

### Option 3: OS-Native Per-Platform (Keychain + secret-tool + wincred)

Implement native secret storage for each OS: macOS Keychain, Linux GNOME Keyring via `secret-tool`,
Windows Credential Manager via `wincred`.

**Pros:** Fully native on every platform.
**Cons:** Triple the implementation and testing surface. GNOME Keyring requires D-Bus and a running
desktop session (fails in headless CI). Windows support isn't needed (Spinner doesn't target Windows).
Significantly more complexity for marginal benefit.

### Option 4: External Tool Integration (`pass`, `1password-cli`, `vault`)

Delegate to an external secret manager chosen by the user.

**Pros:** Rich features, team sharing, audit trails.
**Cons:** External dependency the user must install and configure. Different tools have different CLIs
and auth models. Too much configuration burden for a developer CLI tool.

### Recommendation

**Combine Option 1 (Keychain) and Option 2 (Encrypted File).**

Rationale:
- **Most Spinner users are on macOS** — Keychain gives them the best experience (no passphrase prompts,
  native integration, zero configuration).
- **Linux and CI need a fallback** — the encrypted file backend provides cross-platform coverage with
  strong cryptography and no external dependencies.
- **Auto-detection is simple** — check `runtime.GOOS == "darwin"` and `exec.LookPath("security")`
  succeeds → Keychain; otherwise → encrypted file.
- **Both backends implement the same `Store` interface** — consumers (resolver, CLI commands) are
  backend-agnostic. Adding new backends later (GCP Secret Manager, etc.) is trivial.
- **No CGo** — both backends are pure Go (Keychain via CLI exec, encrypted file via stdlib + x/crypto).

---

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|---|---|---|
| `internal/secret/store.go` | **create** | Store interface, ErrNotFound sentinel error |
| `internal/secret/keychain.go` | **create** | KeychainStore — macOS Keychain via `security` CLI |
| `internal/secret/keychain_test.go` | **create** | Tests with injectable command runner (no real Keychain) |
| `internal/secret/encrypted.go` | **create** | EncryptedFileStore — AES-256-GCM + Argon2id |
| `internal/secret/encrypted_test.go` | **create** | Round-trip, corruption, wrong-passphrase tests |
| `internal/secret/detect.go` | **create** | `NewStore()` auto-detection (darwin → Keychain, else → file) |
| `internal/secret/detect_test.go` | **create** | Platform detection tests |
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

Follows the project's pattern of small focused interfaces (like `Provider`). The `ErrNotFound`
sentinel error lets callers distinguish "not found" from backend failures, matching how
`environmentVariableError` is already used in `prerequisites.go`.

#### Keychain Backend

```go
type KeychainStore struct {
    service string          // Keychain service name, default: "spinner"
    runCmd  commandRunner   // injectable for testing
}
```

Uses `exec.CommandContext` to call `/usr/bin/security`:
- `Set`: `security add-generic-password -U -a spinner -s <key> -w <value>` (`-U` updates if exists)
- `Get`: `security find-generic-password -a spinner -s <key> -w` (outputs password to stdout)
- `Delete`: `security delete-generic-password -a spinner -s <key>`
- `List`: `security dump-keychain` filtered by service name, parse `svce` and `acct` attributes

The `commandRunner` function field defaults to `exec.Command(...).Output()` in production but can be
swapped in tests. This mirrors how the Docker backend mocks its client interface.

#### Encrypted File Backend

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
- **File permissions:** `0600` (owner read/write only), consistent with Docker's temp env file pattern
- **Operations:** All operations load-decrypt-modify-encrypt-write atomically

#### Auto-Detection

```go
func NewStore() (Store, error) {
    // Check override via SPINNER_SECRET_BACKEND env var or Viper config
    if override := viper.GetString("secret-backend"); override != "" {
        // return based on override value ("keychain" or "file")
    }
    // Auto-detect
    if runtime.GOOS == "darwin" {
        if _, err := exec.LookPath("security"); err == nil {
            return NewKeychainStore(), nil
        }
    }
    return NewEncryptedFileStore()
}
```

#### Secret Resolver

```go
type ResolvedSecrets struct {
    Tokens map[string]string  // GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN
    Custom map[string]string  // --secret keys
}

func Resolve(store Store, customKeys []string) (*ResolvedSecrets, error)
```

Resolution order per key:
1. `store.Get(key)` — if found, use it
2. `os.Getenv(key)` — for built-in tokens only (env fallback for backward compat)
3. For built-in tokens: error if neither source has a value
4. For custom `--secret` keys: must exist in store (no env fallback — if user says `--secret NPM_TOKEN`,
   it must be in the store; silent env fallback would undermine the security intent)

#### Spin Command Integration

The resolver centralizes token resolution, replacing the three scattered `os.Getenv` call sites:
- `prerequisites.CheckEnvironmentVariables()` (removed — resolver subsumes this)
- `docker/run.go` line 120-121 (reads from `config.Secrets` instead)
- `gcp_provider.go` line 185-186 (reads from `config.Secrets` instead)

Resolved values are placed in `CreateConfig.Secrets` so backends receive pre-resolved values via
config rather than calling `os.Getenv()` themselves. This matches the existing `EnvVars` pattern
and keeps backends ignorant of the secret storage mechanism.

#### Container Delivery (Unchanged)

- **Docker:** Secrets from `config.Secrets` are written to the temp env file alongside other vars
  (same `0600` permissions, same `defer os.Remove()` cleanup)
- **GCP:** Secrets from `config.Secrets` are written to instance metadata (same as existing tokens)

No changes to the container delivery mechanism. The security improvement is on the host side
(secrets in Keychain/encrypted file instead of plaintext `.envrc`).

### Key Decisions

| Decision | Rationale |
|---|---|
| Secrets in `CreateConfig` (not Store in backends) | Backends stay ignorant of secret storage. Resolution happens once at command level. Matches existing `EnvVars` pattern |
| `--secret` separate from `--env` | `--env KEY=VALUE` exposes value on CLI (shell history). `--secret KEY` references store only (value never on command line). Different security semantics |
| `security` CLI (not CGo Keychain bindings) | No CGo dependency, simpler cross-compilation, stable across macOS versions. Matches Docker backend's exec pattern |
| Argon2id + AES-256-GCM for encrypted file | Current best practice for password-based key derivation + authenticated encryption |
| No env fallback for custom `--secret` keys | If user says `--secret NPM_TOKEN`, they expect it from the secure store. Silent env fallback would undermine security intent. Built-in tokens get env fallback for backward compat only |
| `~/.spinner/secrets.enc` file location | Consistent with existing `~/.spinner/` config directory. `0600` permissions match Docker temp file pattern |
| `SPINNER_SECRET_PASSPHRASE` env var for CI | CI environments cannot use interactive prompts or Keychain. Env var for passphrase is the standard CI escape hatch; alternatively CI users just set `GITHUB_TOKEN` / `CLAUDE_CODE_OAUTH_TOKEN` as env vars directly (existing workflow, no change needed) |

### Dependencies

- `golang.org/x/crypto v0.45.0` (already indirect in go.mod) — for `argon2` package
- `golang.org/x/term v0.37.0` (already indirect in go.mod) — for `ReadPassword` (hidden passphrase input)

No new external modules required. Both are promoted from indirect to direct usage.
