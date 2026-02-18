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
| `internal/provider/provider.go` | **modify** | Add `SecretBlob []byte` + `SecretKey []byte` to `CreateConfig`, remove `Passphrase` and direct token access |
| `internal/backend/docker/run.go` | **modify** | Mount blob + key file; remove env-file token writing; no passphrase in env |
| `internal/backend/docker/run_test.go` | **modify** | Update for blob + key file delivery |
| `internal/backend/docker/docker_provider.go` | **modify** | Map `CreateConfig.SecretBlob` + `SecretKey` → `spinConfig` |
| `internal/backend/gcp/gcp_provider.go` | **modify** | Base64-encode blob as metadata; upload key to GCS (not metadata) |
| `internal/backend/gcp/templates/scripts/gcp_runtime.sh` | **modify** | Fetch key from GCS; write to `/run/spinner/secrets.key` |
| `internal/exec/loop.go` | **modify** | Decrypt blob using key file at startup, inject into executor config |
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
key, blob, err := secret.EncryptBlobWithKey(resolved)
createConfig := provider.CreateConfig{
    // ...existing fields...
    SecretBlob: blob,
    SecretKey:  key,
}
```

#### Container Delivery (Encrypted Blob + Ephemeral Key)

**No secrets or decryption keys are passed as environment variables or instance metadata.** All
secrets are delivered as an encrypted blob. The decryption key is delivered via a file-based side
channel, never through env vars, Docker env-files, or GCP instance metadata.

**Blob Generation (host side, at spin-time):**

```go
// In cmd/spin.go RunE, after Resolve():
key, blob, err := secret.EncryptBlobWithKey(resolved)
createConfig.SecretBlob = blob  // backends mount/upload the blob
createConfig.SecretKey = key    // backends deliver key via file side-channel
```

The blob uses AES-256-GCM with a random 32-byte key (no Argon2id — key derivation is unnecessary
for random keys). A fresh key and nonce are generated per `spin` invocation.

**Docker Backend:**

1. Host writes encrypted blob to `~/.spinner/<container>/secrets.enc` (alongside existing state dir)
2. Host writes ephemeral key to `~/.spinner/<container>/secrets.key`
3. Both mounted read-only into container at `/run/spinner/secrets.enc` and `/run/spinner/secrets.key`
4. No passphrase or key in env-file — nothing secret in `docker inspect`

**GCP Backend:**

1. Host base64-encodes the encrypted blob and passes it as instance metadata key `SPINNER_SECRET_BLOB`
2. Host uploads the ephemeral key to GCS: `gs://<state-bucket>/<instance>/secrets.key`
3. Startup script decodes blob from metadata, writes to `/run/spinner/secrets.enc`
4. Startup script fetches key from GCS via `gcloud storage cp`, writes to `/run/spinner/secrets.key`
5. Key never appears in instance metadata — not visible in GCP Console or metadata API
6. Key persists in GCS for VM restart support (startup script re-reads on each boot)
7. Blob + VM are destroyed on `Remove()` — key in GCS should be cleaned up too

**Why GCS for the key (not metadata):** Instance metadata is visible to anyone with
`compute.instances.get` permission — including the GCP Console VM details page (as seen in the
screenshot that motivated this change). GCS objects require separate `storage.objects.get`
permission and are not surfaced in the VM details UI. The state bucket already exists for logs
and state, so no new infrastructure is needed.

#### `startup.sh` Refactor

The startup script no longer reads `GITHUB_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` as environment
variables. Instead it uses `spinner secret inject` to decrypt the blob for token-dependent work:

```bash
#!/bin/bash
set -e

# Ephemeral key at /run/spinner/secrets.key (mounted file or fetched from GCS).
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
keyPath := "/run/spinner/secrets.key"
if fileExists(blobPath) && fileExists(keyPath) {
    secrets, err := secret.DecryptBlobWithKeyFile(blobPath, keyPath)
    // DO NOT delete blob or key — needed for inception scenarios
    // Inject all secrets + key/blob paths into child process env for inception
    envSlice := secretsToEnvSlice(secrets)
    envSlice = append(envSlice, "SPINNER_SECRET_STORE=/run/spinner/secrets.enc")
    envSlice = append(envSlice, "SPINNER_SECRET_KEY=/run/spinner/secrets.key")
    executorConfig.Env = append(executorConfig.Env, envSlice...)
}
```

1. Read blob from `/run/spinner/secrets.enc`
2. Read key from `/run/spinner/secrets.key`
3. Decrypt secrets into memory
4. **Keep blob and key on disk** — needed for inception (inner `spinner spin`)
5. Inject secrets + `SPINNER_SECRET_STORE` + `SPINNER_SECRET_KEY` via `cmd.Env` when spawning
   Claude CLI (existing mechanism at `executor.go:83-85`)

The key file path is included in child process env because it's **redundant information** — the
agent already has every decrypted secret value. But including it enables the agent to run inception
(`spinner spin --secret ...`) without user intervention. The blob and key stay on disk so the inner
spinner can read from them via `SPINNER_SECRET_STORE` + `SPINNER_SECRET_KEY`.

#### No-`--prompt` Mode: `spinner secret inject`

When user SSHs into the container:

1. Blob exists at `/run/spinner/secrets.enc` (encrypted, unreadable without key)
2. Key exists at `/run/spinner/secrets.key` (mounted file, `0600` permissions)
3. No passphrase in container env — nothing in `docker inspect` or instance metadata
4. User runs `spinner secret inject -- <command>` to access secrets:

```bash
# Decrypt and inject for a specific command
spinner secret inject -- claude -p "implement feature X"

# Or start a subshell with secrets available
spinner secret inject -- bash

# Inception: run inner spinner with secrets from outer blob
spinner secret inject -- sh -c '
  SPINNER_SECRET_STORE=/run/spinner/secrets.enc \
  SPINNER_SECRET_KEY=/run/spinner/secrets.key \
  spinner spin --backend docker --secret NPM_TOKEN --repo ... --prompt "task"
'
```

`spinner secret inject` implementation:

```go
// In cmd/secret.go:
func runSecretInject(cmd *cobra.Command, args []string) error {
    keyPath := keyPathFromEnvOrDefault()  // SPINNER_SECRET_KEY env, or /run/spinner/secrets.key
    secrets, err := secret.DecryptBlobWithKeyFile(blobPath, keyPath)
    // Run the command with secrets injected
    child := exec.Command(args[0], args[1:]...)
    child.Env = append(os.Environ(), secretsToEnvSlice(secrets)...)
    child.Stdin = os.Stdin
    child.Stdout = os.Stdout
    child.Stderr = os.Stderr
    return child.Run()
}
```

**Note:** `inject` reads the key from `SPINNER_SECRET_KEY` path (env var pointing to file), falling
back to `/run/spinner/secrets.key`. Both Docker and GCP place the key file at that default path.
If the key file is missing (e.g., user SSHed into a container created before this change), `inject`
falls back to prompting for a passphrase and using the Argon2id path for backward compatibility.

### Key Decisions

| Decision | Rationale |
|---|---|
| Encrypted file only (no Keychain) | Single codepath, cross-platform, pure Go. Tokens are pass-through so Keychain sophistication isn't needed |
| **No env var fallback (breaking change)** | Eliminates plaintext `.envrc` workflow entirely. No users yet, clean migration |
| **All tokens in blob (no split)** | GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN, and custom secrets all travel the same way. No special-casing |
| `--secret` separate from `--env` | `--env KEY=VALUE` exposes value on CLI. `--secret KEY` references store only. Different security semantics |
| Argon2id + AES-256-GCM for host store | Current best practice for password-based key derivation + authenticated encryption |
| **Random key + raw AES-256-GCM for transport blob** | No Argon2id needed — random 32-byte key is already full entropy. Faster, simpler. |
| **Key via file, not env/metadata** | Passphrase in GCP metadata was visible in Console UI. Key-as-file eliminates env var and metadata leaks entirely. |
| **GCS for GCP key delivery** | State bucket already exists. GCS IAM is separate from `compute.instances.get`. Key not visible in VM details page. |
| **Key persists (not deleted after read)** | VM restarts re-run startup script. Key must be re-readable. GCS persistence mirrors metadata persistence model. |
| `~/.spinner/secrets.enc` location | Consistent with existing `~/.spinner/` config directory |
| `SPINNER_SECRET_PASSPHRASE` env var (host only) | Standard CI escape hatch for unlocking the host store. Never sent to containers. |
| Encrypted blob (not env vars) for ALL secrets | No secret values in container env, `ps aux`, or Docker env-file |
| **Separate key for transport blob** | Host store uses user passphrase + Argon2id. Transport blob uses random key. Passphrase never leaves host. |
| **Blob + key retained on disk (not deleted)** | Enables inception scenarios and VM restarts — inner spinner reads from outer blob + key |
| **Key file path forwarded to child processes** | Via `SPINNER_SECRET_KEY` env var. Redundant info (agent has all secrets). Enables inception without user intervention. |
| `spinner secret inject` wrapper (not global export) | Limits secret exposure to explicit command trees. User controls which processes get secrets |
| **startup.sh uses `spinner secret inject`** | Git credential cache persists auth. Tokens as env vars are redundant after initial setup |

#### Inception: Spinner Inside Spinner

A common pattern is: local machine → GCP VM → Docker containers inside the VM. Each layer needs
access to secrets without the user's host store being available.

The encrypted blob solves this because **blob format = store format**. The store path and key path
are configurable via `SPINNER_SECRET_STORE` and `SPINNER_SECRET_KEY` environment variables. At each
layer:

```
Layer 0 (local machine):
  spinner spin --backend gcp --secret NPM_TOKEN --secret API_KEY --prompt "task"
  → reads from ~/.spinner/secrets.enc (host store, passphrase-protected)
  → generates random key + encrypted blob
  → blob → instance metadata, key → GCS
  → VM gets /run/spinner/secrets.enc + /run/spinner/secrets.key

Layer 1 (GCP VM, --prompt mode):
  spinner exec reads blob + key → decrypts → injects into Claude CLI
  → child process has: NPM_TOKEN, API_KEY, GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN,
    SPINNER_SECRET_STORE=/run/spinner/secrets.enc,
    SPINNER_SECRET_KEY=/run/spinner/secrets.key
  → if agent runs: spinner spin --backend docker --secret NPM_TOKEN --prompt "sub-task"
    → inner spinner reads from /run/spinner/secrets.enc (blob = store, key = key)
    → generates new random key + blob → mounts into Docker container

Layer 1 (GCP VM, user SSHs in):
  spinner secret inject -- sh -c '
    SPINNER_SECRET_STORE=/run/spinner/secrets.enc \
    SPINNER_SECRET_KEY=/run/spinner/secrets.key \
    spinner spin --backend docker --secret NPM_TOKEN --prompt "task"
  '
  → key file exists at /run/spinner/secrets.key, no passphrase needed
  → inner spinner reads from blob, generates new key + blob for inner container

Layer 2 (Docker container, --prompt mode):
  spinner exec reads inner blob + key → decrypts → injects into Claude CLI
```

Same pattern at every layer — key file + blob file. The passphrase never leaves the host. Each
layer decrypts with its key, re-encrypts with a fresh random key for the next. No store setup
required inside VMs or containers — the blob IS the store.

**Implementation:** `EncryptedFileStore` already takes a configurable `path`. The additions are
`SPINNER_SECRET_KEY` for the key file path and `EncryptBlobWithKey`/`DecryptBlobWithKeyFile` for
raw-key operations:

```go
func defaultStorePath() string {
    if p := os.Getenv("SPINNER_SECRET_STORE"); p != "" {
        return p
    }
    return filepath.Join(home, ".spinner", "secrets.enc")
}

func defaultKeyPath() string {
    if p := os.Getenv("SPINNER_SECRET_KEY"); p != "" {
        return p
    }
    return "/run/spinner/secrets.key"
}
```

### Dependencies

- `golang.org/x/crypto v0.45.0` (already indirect in go.mod) — for `argon2` package
- `golang.org/x/term v0.37.0` (already indirect in go.mod) — for `ReadPassword` (hidden passphrase input)

No new external modules required. Both are promoted from indirect to direct usage.
