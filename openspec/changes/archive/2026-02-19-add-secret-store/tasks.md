# Tasks: Add Secret Store

## 1.0 Store Interface + Encrypted File Backend

- [x] 1.1 Create `internal/secret/store.go` with `Store` interface (`Set`, `Get`, `Delete`, `List`) and `ErrNotFound` sentinel error
- [x] 1.2 Create `internal/secret/encrypted.go` with `EncryptedFileStore` implementation (AES-256-GCM + Argon2id, configurable path via `SPINNER_SECRET_STORE` env var defaulting to `~/.spinner/secrets.enc`, atomic writes, `0600` permissions, injectable passphrase function)
- [x] 1.3 Create `internal/secret/encrypted_test.go` with tests: round-trip set/get, multiple keys, delete, list, wrong passphrase error, corrupted file error, missing file returns empty store, atomic write safety, custom store path via env var
- [x] 1.4 Create `internal/secret/mock_store.go` with testify `MockStore` for consumer tests
- [x] 1.5 Verify build and all tests pass

## 2.0 `spinner secret` CLI Subcommand

- [x] 2.1 Add `flagSecret` constant to `cmd/helpers.go`
- [x] 2.2 Create `cmd/secret.go` with `spinner secret` parent command and `set`, `list`, `delete` subcommands using Store interface injection
- [x] 2.3 Create `cmd/secret_test.go` with MockStore-based tests: set (prompted and `--value`), list, delete, delete nonexistent key error
- [x] 2.4 Verify build and all tests pass

## 3.0 Encrypted Blob Transport

- [x] 3.1 Create `internal/secret/blob.go` with `EncryptBlob(secrets map[string]string, passphrase string) ([]byte, error)` and `DecryptBlob(path string, passphrase string) (map[string]string, error)` — same AES-256-GCM + Argon2id scheme as store, fresh salt per blob
- [x] 3.2 Create `internal/secret/blob_test.go` with tests: round-trip encrypt/decrypt, wrong passphrase error, corrupted blob error, empty secrets map
- [x] 3.3 Verify build and all tests pass

## ~~4.0 Secret Resolver + Spin Command Integration~~

- [x] 4.1 Create `internal/secret/resolver.go` with `Resolve(store, customKeys)` function: store-only resolution for ALL tokens (built-in + custom), no env fallback, error with "run spinner secret set" suggestion on missing key
- [x] 4.2 Create `internal/secret/resolver_test.go` with tests: token from store, missing token error (no env fallback), custom key from store, custom key not found error, error message includes "spinner secret set" suggestion
- [x] 4.3 Add `SecretBlob []byte` and `Passphrase string` to `provider.CreateConfig`, remove direct token access patterns
- [x] 4.4 Modify `cmd/spin.go`: add `--secret` flag (StringSliceVar), create Store via injectable factory, call Resolve (all tokens from store, no env fallback), generate encrypted blob from ALL resolved secrets, populate `CreateConfig.SecretBlob` and `CreateConfig.Passphrase`
- [x] 4.5 Modify `internal/backend/docker/run.go`: remove env-file token writing (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`); write encrypted blob to `~/.spinner/<container>/secrets.enc`; mount blob read-only at `/run/spinner/secrets.enc`; pass `SPINNER_SECRET_PASSPHRASE` as container env var
- [x] 4.6 Modify `internal/backend/docker/docker_provider.go`: pass `SecretBlob` and `Passphrase` from `CreateConfig` to `spinConfig`
- [x] 4.7 Modify `internal/backend/gcp/gcp_provider.go`: remove metadata token writing; base64-encode encrypted blob and pass as `SPINNER_SECRET_BLOB` metadata key; pass `SPINNER_SECRET_PASSPHRASE` as metadata; update `updateMetadata` to refresh blob/passphrase on restart
- [x] 4.8 Remove `prerequisites.CheckEnvironmentVariables()` function (replaced by resolver)
- [x] 4.9 Update all affected tests: `cmd/spin_test.go`, `cmd/helpers_test.go`, `docker/run_test.go`, `gcp/gcp_provider_test.go`, `prerequisites/prerequisites_test.go`
- [x] 4.10 Verify build and all tests pass

## 5.0 Startup Script Refactor

- [x] 5.1 Modify `templates/scripts/startup.sh`: remove `GITHUB_TOKEN` env var check; use `spinner secret inject -- sh -c '...'` to wrap `gh auth setup-git`, credential cache config, and `git clone`/fetch; keep branch checkout and `spinner exec`/`tail -f` outside the inject wrapper (git credentials are cached)
- [x] 5.2 Add error handling for missing secrets blob in startup.sh
- [x] 5.3 Verify build and all tests pass

## ~~6.0 Spinner Exec Secret Injection~~

- [x] 6.1 Modify `internal/exec/loop.go`: at startup, check for `/run/spinner/secrets.enc` + `SPINNER_SECRET_PASSPHRASE`; decrypt blob into memory; DO NOT delete blob (retained for inception); unset passphrase from own env; inject decrypted secrets + `SPINNER_SECRET_PASSPHRASE` + `SPINNER_SECRET_STORE=/run/spinner/secrets.enc` into executor config Env
- [x] 6.2 Verify `internal/agent/claude/executor.go` already injects `config.Env` via `cmd.Env` (no change needed)
- [x] 6.3 Add tests to `internal/exec/loop_test.go`: secrets blob decrypted and injected, passphrase forwarded to child env, SPINNER_SECRET_STORE set in child env, blob NOT deleted, passphrase unset from own env, missing blob continues normally, corrupted blob logs warning and continues
- [x] 6.4 Verify build and all tests pass

## ~~7.0 Secret Inject Command (In-Container)~~

- [x] 7.1 Add `spinner secret inject -- <command>` subcommand to `cmd/secret.go`: read passphrase from `SPINNER_SECRET_PASSPHRASE` env first then interactive prompt, decrypt blob at `/run/spinner/secrets.enc`, run command with secrets as env vars, exit with child's exit code
- [x] 7.2 Add tests to `cmd/secret_test.go`: inject decrypts and runs command, passphrase from env, passphrase from prompt, wrong passphrase error, missing blob error, missing command argument error
- [x] 7.3 Verify build and all tests pass

## ~~8.0 Documentation~~

- [x] 8.1 Update `docs/usage.md` with secret management workflow: `spinner secret set/list/delete`, `--secret` flag on spin, encrypted blob delivery, `spinner secret inject` for in-container use, inception scenarios, breaking change migration (env vars → store)

## ~~9.0 Raw-Key Blob Encryption Functions~~

- [x] 9.1 Add `EncryptBlobWithKey(secrets) (key, blob, error)` and `DecryptBlobWithKeyFile(blobPath, keyPath) (secrets, error)` to `internal/secret/blob.go` — random 32-byte AES-256-GCM key (no Argon2id), nonce + ciphertext format
- [x] 9.2 Add tests to `internal/secret/blob_test.go`: round-trip with key file, wrong key error, missing key file error, fresh key per call
- [x] 9.3 Verify build and all tests pass

## ~~10.0 Switch Transport from Passphrase to Key-File~~

- [x] 10.1 Replace `Passphrase string` with `SecretKey []byte` in `provider.CreateConfig` and update doc comment; update `spinConfig` in Docker backend
- [x] 10.2 Update `cmd/spin.go`: use `EncryptBlobWithKey` instead of `EncryptBlob`, populate `SecretKey` instead of `Passphrase`, remove passphrase retrieval for blob
- [x] 10.3 Update Docker backend `run.go`: write key file to `~/.spinner/<container>/secrets.key`, mount at `/run/spinner/secrets.key:ro`, remove `SPINNER_SECRET_PASSPHRASE` from env-file; update `docker_provider.go` to pass `SecretKey`
- [x] 10.4 Update GCP backend `gcp_provider.go`: upload key to GCS at `gs://<bucket>/<instance>/secrets.key` via `UploadObject`, remove `SPINNER_SECRET_PASSPHRASE` from metadata; update `updateMetadata` to re-upload key on Start; update `gcp_runtime.sh` to fetch key from GCS
- [x] 10.5 Update `internal/exec/loop.go`: use key file (`/run/spinner/secrets.key`) instead of `SPINNER_SECRET_PASSPHRASE` env; inject `SPINNER_SECRET_KEY` instead of `SPINNER_SECRET_PASSPHRASE` into child env; remove `osUnsetenv` call
- [x] 10.6 Update `cmd/secret.go` inject command: read key from `SPINNER_SECRET_KEY` env or `/run/spinner/secrets.key` default; fall back to passphrase prompt if no key file
- [x] 10.7 Add `SPINNER_SECRET_PASSPHRASE` and `SPINNER_SECRET_KEY` to reserved env vars in `cmd/spin.go`
- [x] 10.8 Update all affected tests: `blob_test.go`, `run_test.go`, `gcp_provider_test.go`, `loop_test.go`, `secret_test.go`, `spin_test.go`
- [x] 10.9 Verify build and all tests pass
