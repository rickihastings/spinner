# Design: Add Secret Store

## Key Insight: Tokens Are Pass-Through

Spinner never uses `GITHUB_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` on the host. It reads them and
forwards them into containers:

- `internal/backend/docker/run.go:120-121` — writes to temp env file
- `internal/backend/gcp/gcp_provider.go:185-186` — writes to instance metadata
- `internal/prerequisites/prerequisites.go:20-33` — validates non-empty

The host CLI is a pass-through. This means the secret store only needs to provide values at
spin-time — unlock once, read values, encrypt into a blob, forward into the container. No persistent
runtime access, no background processes, no daemon.

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
  prompt for local development.
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
| `internal/secret/resolver.go` | **create** | `Resolve(store, customKeys)` — store only, no env fallback |
| `internal/secret/resolver_test.go` | **create** | Resolution order and error condition tests |
| `internal/secret/mock_store.go` | **create** | Testify MockStore for consumer tests |
| `cmd/secret.go` | **create** | `spinner secret` subcommand (set/list/delete/inject) |
| `cmd/secret_test.go` | **create** | Subcommand tests with MockStore injection |
| `cmd/helpers.go` | **modify** | Add `flagSecret` constant |
| `cmd/spin.go` | **modify** | Add `--secret` flag, create Store, resolve all secrets, generate blob |
| `cmd/spin_test.go` | **modify** | Test `--secret` flag parsing and validation |
| `internal/provider/provider.go` | **modify** | Add `SecretBlob []byte` to `CreateConfig`, remove direct token access |
| `internal/backend/docker/run.go` | **modify** | Mount blob; remove env-file token writing; pass `SPINNER_SECRET_PASSPHRASE` as sole env var |
| `internal/backend/docker/run_test.go` | **modify** | Update for blob-based delivery |
| `internal/backend/docker/docker_provider.go` | **modify** | Map `CreateConfig.SecretBlob` → `spinConfig` |
| `internal/backend/gcp/gcp_provider.go` | **modify** | Base64-encode blob as `SPINNER_SECRET_BLOB` metadata; pass `SPINNER_SECRET_PASSPHRASE` as metadata |
| `internal/exec/loop.go` | **modify** | Decrypt blob at startup, inject into executor config (including passphrase for inception) |
| `internal/prerequisites/prerequisites.go` | **modify** | Remove `CheckEnvironmentVariables()` (replaced by resolver) |
| `templates/scripts/startup.sh` | **modify** | Refactor to use `spinner secret inject` for token-dependent work |
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
    path       string                  // default: ~/.spinner/secrets.enc (overridable via SPINNER_SECRET_STORE)
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
- **Store path:** Configurable via `SPINNER_SECRET_STORE` env var, defaults to `~/.spinner/secrets.enc`

The store is unlocked once per CLI invocation (the passphrase function is called on first
access). Since tokens are pass-through, there's only one unlock per `spin` command.

#### Secret Resolver

```go
func Resolve(store Store, customKeys []string) (map[string]string, error)
```

Returns a single `map[string]string` containing all resolved secrets (built-in tokens + custom keys).

**All keys resolve from the store only. No environment variable fallback.**

Resolution per key:
1. `store.Get(key)` — if found, use it
2. If not found: error with message suggesting `spinner secret set <KEY>`

Built-in tokens (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`) are treated identically to custom keys.
They must exist in the store. This is a **breaking change** from the previous env-var workflow.

The resolver replaces the three scattered `os.Getenv` call sites:
- `prerequisites.CheckEnvironmentVariables()` — removed, resolver subsumes this
- `docker/run.go:120-121` — removed, tokens travel via blob
- `gcp_provider.go:185-186` — removed, tokens travel via blob

#### Spin Command Integration

Resolved values are encrypted into a blob. Backends receive the blob for mounting/delivery.

```go
// In cmd/spin.go RunE:
store := secret.NewEncryptedFileStore(defaultStorePath(), passphraseFunc)
resolved, err := secret.Resolve(store, spinSecrets)  // spinSecrets from --secret flags
blob, err := secret.EncryptBlob(resolved, passphrase)
createConfig := provider.CreateConfig{
    // ...existing fields...
    SecretBlob: blob,
}
```

#### Container Delivery (Encrypted Blob)

**No secrets are passed as container environment variables.** All secrets (built-in tokens and
custom) are delivered as an encrypted blob that requires explicit decryption inside the container.
The only env var/metadata passed is `SPINNER_SECRET_PASSPHRASE` — the decryption key.

**Blob Generation (host side, at spin-time):**

```go
// In cmd/spin.go RunE, after Resolve():
blob, err := secret.EncryptBlob(resolved, passphrase)
createConfig.SecretBlob = blob  // backends mount/upload this
```

The blob uses the same AES-256-GCM + Argon2id scheme as the host store but with a fresh salt.
The user's store passphrase encrypts the blob — same passphrase, different salt, separate file.

**Docker Backend:**

1. Host writes encrypted blob to `~/.spinner/<container>/secrets.enc` (alongside existing state dir)
2. Mounted read-only into container at `/run/spinner/secrets.enc` via `-v` flag
3. `SPINNER_SECRET_PASSPHRASE` passed as container env var (both modes — startup.sh needs it)

**GCP Backend:**

1. Host base64-encodes the encrypted blob and passes it as instance metadata key `SPINNER_SECRET_BLOB`
2. Startup script decodes the metadata value and writes it to `/run/spinner/secrets.enc`
3. `SPINNER_SECRET_PASSPHRASE` passed as instance metadata key (both modes — startup script needs it)
4. Blob is destroyed when the VM is deleted — no orphaned files in GCS

**Passphrase in container env:** `SPINNER_SECRET_PASSPHRASE` is always passed to the container because
`startup.sh` needs it to decrypt the blob for initial git auth and clone. It remains discoverable via
`/proc/1/environ` (Docker) or metadata API (GCP). This is defense in depth — secrets are not casually
visible via `env` but a determined process can find the passphrase. This is an accepted tradeoff.

#### `startup.sh` Refactor

The startup script no longer reads `GITHUB_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` as environment
variables. Instead it uses `spinner secret inject` to decrypt the blob for token-dependent work:

```bash
#!/bin/bash
set -e

# SPINNER_SECRET_PASSPHRASE is set in container env / instance metadata.
# All secrets are in the encrypted blob at /run/spinner/secrets.enc.
# Use `spinner secret inject` to decrypt and run token-dependent commands.

if [ -z "$REPO_URL" ]; then
  echo "Error: REPO_URL environment variable is not set"
  exit 1
fi

# Decrypt secrets and run git auth setup + clone in one shot.
# gh auth setup-git configures git credential helper.
# Credential cache (1-year timeout) persists auth after this block exits.
# After this, GITHUB_TOKEN is no longer needed as an env var for git operations.
spinner secret inject -- sh -c '
  gh auth setup-git
  git config --global credential.helper "cache --timeout=31536000"
  if [ -d ".git" ]; then
    CURRENT_REMOTE=$(git config --get remote.origin.url || echo "")
    if [ "$CURRENT_REMOTE" != "'"$REPO_URL"'" ]; then
      echo "Error: Existing repo URL ($CURRENT_REMOTE) does not match expected ($REPO_URL)"
      exit 1
    fi
    git fetch origin
  else
    git clone "$REPO_URL" .
  fi
'

git status

# If PROMPT is set, run the iteration loop
if [ -n "$PROMPT" ]; then
  # Branch checkout...
  DEFAULT_BRANCH=$(git symbolic-ref refs/remotes/origin/HEAD | sed 's@^refs/remotes/origin/@@')
  if [ -n "$BRANCH" ]; then
    BRANCH="${BRANCH#\'}"
    BRANCH="${BRANCH%\'}"
    if git rev-parse --verify "$BRANCH" >/dev/null 2>&1; then
      git checkout "$BRANCH"
    elif git ls-remote --heads origin "$BRANCH" | grep -q "$BRANCH"; then
      git checkout "$BRANCH"
    else
      git checkout -b "$BRANCH"
    fi
  fi

  echo "Starting autonomous implementation loop..."
  spinner exec
else
  echo "Repository cloned successfully. Container is ready."
  tail -f /dev/null
fi
```

**Key insight:** `gh auth setup-git` configures git to use `gh auth git-credential` as a credential
helper. Combined with `git config credential.helper 'cache --timeout=31536000'`, the credentials are
cached for 1 year. After the initial clone/fetch, `GITHUB_TOKEN` is no longer needed as an
environment variable for git operations. The token IS still needed by:
- `gh` CLI commands (e.g., `gh pr create`) — these run inside `spinner exec` child processes or
  `spinner secret inject` wrappers, which inject it from the blob
- Inner `spinner spin` commands (inception) — which read from the blob via `SPINNER_SECRET_STORE`

#### `--prompt` Mode: `spinner exec` as Secret Broker

When `spinner exec` starts inside the container:

```go
// In internal/exec/loop.go, before entering iteration loop:
blobPath := "/run/spinner/secrets.enc"
passphrase := os.Getenv("SPINNER_SECRET_PASSPHRASE")
if passphrase != "" && fileExists(blobPath) {
    secrets, err := secret.DecryptBlob(blobPath, passphrase)
    // DO NOT delete the blob — needed for inception scenarios
    os.Unsetenv("SPINNER_SECRET_PASSPHRASE")     // remove passphrase from own env
    // Inject all secrets + passphrase into child process env for inception
    envSlice := secretsToEnvSlice(secrets)
    envSlice = append(envSlice, "SPINNER_SECRET_PASSPHRASE="+passphrase)
    envSlice = append(envSlice, "SPINNER_SECRET_STORE=/run/spinner/secrets.enc")
    executorConfig.Env = append(executorConfig.Env, envSlice...)
}
```

1. Read blob from `/run/spinner/secrets.enc`
2. Read passphrase from `SPINNER_SECRET_PASSPHRASE` env var
3. Decrypt secrets into memory
4. **Keep blob on disk** — needed for inception (inner `spinner spin`)
5. Unset `SPINNER_SECRET_PASSPHRASE` from own process environment
6. Inject secrets + `SPINNER_SECRET_PASSPHRASE` + `SPINNER_SECRET_STORE` via `cmd.Env` when spawning
   Claude CLI (existing mechanism at `executor.go:83-85`)

The passphrase is included in child process env because it's **redundant information** — the agent
already has every decrypted secret value. But including it enables the agent to run inception
(`spinner spin --secret ...`) without user intervention. The blob stays on disk so the inner spinner
can read from it via `SPINNER_SECRET_STORE`.

#### No-`--prompt` Mode: `spinner secret inject`

When user SSHs into the container:

1. Blob exists at `/run/spinner/secrets.enc` (encrypted, unreadable without passphrase)
2. `SPINNER_SECRET_PASSPHRASE` is in the container env (Docker `/proc/1/environ`, GCP metadata) —
   used by startup.sh and discoverable by determined processes, but not casually visible via `env`
   in an SSH session
3. User runs `spinner secret inject -- <command>` to access secrets:

```bash
# Decrypt and inject for a specific command
spinner secret inject -- claude -p "implement feature X"

# Or start a subshell with secrets available
spinner secret inject -- bash

# Inception: run inner spinner with secrets from outer blob
spinner secret inject -- sh -c '
  SPINNER_SECRET_STORE=/run/spinner/secrets.enc \
  spinner spin --backend docker --secret NPM_TOKEN --repo ... --prompt "task"
'
```

`spinner secret inject` implementation:

```go
// In cmd/secret.go:
func runSecretInject(cmd *cobra.Command, args []string) error {
    passphrase := getPassphrase()  // SPINNER_SECRET_PASSPHRASE env, then interactive prompt
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

**Note:** `inject` reads the passphrase from `SPINNER_SECRET_PASSPHRASE` env var first, then falls
back to interactive prompt. In Docker, the container env var is set (from `--env-file` at creation),
so `inject` can decrypt non-interactively. In GCP, the startup script can write it to a restricted
file. In both cases, the user CAN also type it manually.

### Key Decisions

| Decision | Rationale |
|---|---|
| Encrypted file only (no Keychain) | Single codepath, cross-platform, pure Go. Tokens are pass-through so Keychain sophistication isn't needed |
| **No env var fallback (breaking change)** | Eliminates plaintext `.envrc` workflow entirely. No users yet, clean migration |
| **All tokens in blob (no split)** | GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN, and custom secrets all travel the same way. No special-casing |
| `--secret` separate from `--env` | `--env KEY=VALUE` exposes value on CLI. `--secret KEY` references store only. Different security semantics |
| Argon2id + AES-256-GCM | Current best practice for password-based key derivation + authenticated encryption |
| `~/.spinner/secrets.enc` location | Consistent with existing `~/.spinner/` config directory |
| `SPINNER_SECRET_PASSPHRASE` env var | Standard CI escape hatch on the host. Also used inside containers for startup.sh blob decryption |
| **Passphrase always in container env** | startup.sh needs it for initial git auth. Defense in depth — not casually visible but discoverable |
| Encrypted blob (not env vars) for ALL secrets | No secret values in container env, `ps aux`, or Docker env-file |
| Same passphrase for host store and container blob | Single passphrase UX. Container blob has its own salt. Host store file never enters the container |
| **Blob retained on disk (not deleted)** | Enables inception scenarios — inner spinner reads from outer blob |
| **Passphrase forwarded to child processes** | Redundant info (agent has all secrets). Enables inception without user intervention |
| `spinner secret inject` wrapper (not global export) | Limits secret exposure to explicit command trees. User controls which processes get secrets |
| **startup.sh uses `spinner secret inject`** | Git credential cache persists auth. Tokens as env vars are redundant after initial setup |

#### Inception: Spinner Inside Spinner

A common pattern is: local machine → GCP VM → Docker containers inside the VM. Each layer needs
access to secrets without the user's host store being available.

The encrypted blob solves this because **blob format = store format**. The store path is configurable
via `SPINNER_SECRET_STORE` environment variable (default: `~/.spinner/secrets.enc`). At each layer:

```
Layer 0 (local machine):
  spinner spin --backend gcp --secret NPM_TOKEN --secret API_KEY --prompt "task"
  → reads from ~/.spinner/secrets.enc (host store)
  → generates encrypted blob → passes as instance metadata
  → VM gets /run/spinner/secrets.enc

Layer 1 (GCP VM, --prompt mode):
  spinner exec reads blob → decrypts → injects into Claude CLI
  → child process has: NPM_TOKEN, API_KEY, GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN,
    SPINNER_SECRET_PASSPHRASE, SPINNER_SECRET_STORE=/run/spinner/secrets.enc
  → if agent runs: spinner spin --backend docker --secret NPM_TOKEN --prompt "sub-task"
    → inner spinner reads from /run/spinner/secrets.enc (blob = store)
    → generates new blob → mounts into Docker container

Layer 1 (GCP VM, user SSHs in):
  spinner secret inject -- sh -c '
    SPINNER_SECRET_STORE=/run/spinner/secrets.enc \
    spinner spin --backend docker --secret NPM_TOKEN --prompt "task"
  '
  → user provides passphrase (or it's read from SPINNER_SECRET_PASSPHRASE in env)
  → inner spinner reads from blob, generates new blob for inner container

Layer 2 (Docker container, --prompt mode):
  spinner exec reads inner blob → decrypts → injects into Claude CLI
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
