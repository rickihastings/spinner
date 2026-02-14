# Tasks: Add Secret Store

## 1.0 Store Interface + Encrypted File Backend

- [ ] 1.1 Create `internal/secret/store.go` with `Store` interface (`Set`, `Get`, `Delete`, `List`) and `ErrNotFound` sentinel error
- [ ] 1.2 Create `internal/secret/encrypted.go` with `EncryptedFileStore` implementation (AES-256-GCM + Argon2id, configurable path via `SPINNER_SECRET_STORE` env var defaulting to `~/.spinner/secrets.enc`, atomic writes, `0600` permissions, injectable passphrase function)
- [ ] 1.3 Create `internal/secret/encrypted_test.go` with tests: round-trip set/get, multiple keys, delete, list, wrong passphrase error, corrupted file error, missing file returns empty store, atomic write safety
- [ ] 1.4 Create `internal/secret/mock_store.go` with testify `MockStore` for consumer tests
- [ ] 1.5 Verify build and all tests pass

## 2.0 `spinner secret` CLI Subcommand

- [ ] 2.1 Add `flagSecret` constant to `cmd/helpers.go`
- [ ] 2.2 Create `cmd/secret.go` with `spinner secret` parent command and `set`, `list`, `delete` subcommands using Store interface injection
- [ ] 2.3 Create `cmd/secret_test.go` with MockStore-based tests: set (prompted and `--value`), list, delete, delete nonexistent key error
- [ ] 2.4 Verify build and all tests pass

## 3.0 Encrypted Blob Transport

- [ ] 3.1 Create `internal/secret/blob.go` with `EncryptBlob(secrets map[string]string, passphrase string) ([]byte, error)` and `DecryptBlob(path string, passphrase string) (map[string]string, error)` — same AES-256-GCM + Argon2id scheme as store, fresh salt per blob
- [ ] 3.2 Create `internal/secret/blob_test.go` with tests: round-trip encrypt/decrypt, wrong passphrase error, corrupted blob error, empty secrets map
- [ ] 3.3 Verify build and all tests pass

## 4.0 Secret Resolver + Spin Command Integration

- [ ] 4.1 Create `internal/secret/resolver.go` with `Resolve(store, customKeys)` function: store-first resolution for built-in tokens with env fallback, store-only for custom keys, error on missing required tokens
- [ ] 4.2 Create `internal/secret/resolver_test.go` with tests: token from store, token from env fallback, store overrides env, missing token error, custom key from store, custom key not found error
- [ ] 4.3 Add `Secrets map[string]string` and `SecretBlob []byte` to `provider.CreateConfig`
- [ ] 4.4 Modify `cmd/spin.go`: add `--secret` flag (StringSliceVar), create Store, call Resolve, split built-in tokens from custom secrets, generate encrypted blob for custom secrets, populate `CreateConfig`
- [ ] 4.5 Modify `internal/backend/docker/run.go`: read built-in tokens from `config.Secrets`; write encrypted blob to `~/.spinner/<container>/secrets.enc`; mount blob read-only at `/run/spinner/secrets.enc`; pass `SPINNER_SECRET_PASSPHRASE` as env var only when prompt is set
- [ ] 4.6 Modify `internal/backend/docker/docker_provider.go`: pass `Secrets` and `SecretBlob` from `CreateConfig` to `spinConfig`
- [ ] 4.7 Modify `internal/backend/gcp/gcp_provider.go`: read built-in tokens from `config.Secrets`; base64-encode encrypted blob and pass as `SPINNER_SECRET_BLOB` metadata key; pass `SPINNER_SECRET_PASSPHRASE` as metadata only when prompt is set; update startup script to decode blob to `/run/spinner/secrets.enc`
- [ ] 4.8 Remove `prerequisites.CheckEnvironmentVariables()` function (replaced by resolver)
- [ ] 4.9 Update all affected tests: `cmd/spin_test.go`, `docker/run_test.go`, `docker/docker_provider_test.go`, `gcp/gcp_provider_test.go`, `prerequisites/prerequisites_test.go`
- [ ] 4.10 Verify build and all tests pass

## 5.0 Spinner Exec Secret Injection

- [ ] 5.1 Modify `internal/exec/loop.go`: at startup, check for `/run/spinner/secrets.enc` + `SPINNER_SECRET_PASSPHRASE`; decrypt blob into memory; delete blob file; unset passphrase env var; add decrypted secrets to executor config Env
- [ ] 5.2 Verify `internal/agent/claude/executor.go` already injects `config.Env` via `cmd.Env` (no change needed)
- [ ] 5.3 Add tests to `internal/exec/loop_test.go`: secrets blob decrypted and injected, blob deleted after decryption, passphrase unset, missing blob continues normally, corrupted blob logs warning and continues
- [ ] 5.4 Verify build and all tests pass

## 6.0 Secret Inject Command (In-Container)

- [ ] 6.1 Add `spinner secret inject -- <command>` subcommand to `cmd/secret.go`: prompt for passphrase, decrypt blob at `/run/spinner/secrets.enc`, run command with secrets as env vars, exit with child's exit code
- [ ] 6.2 Add tests to `cmd/secret_test.go`: inject decrypts and runs command, wrong passphrase error, missing blob error, missing command argument error
- [ ] 6.3 Verify build and all tests pass

## 7.0 Documentation

- [ ] 7.1 Update `docs/usage.md` with secret management workflow: `spinner secret set/list/delete`, `--secret` flag on spin, encrypted blob delivery, `spinner secret inject` for in-container use, env var fallback behavior
