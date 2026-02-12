# Tasks: Add Secret Store

## 1.0 Store Interface + Keychain Backend

- [ ] 1.1 Create `internal/secret/store.go` with `Store` interface (`Set`, `Get`, `Delete`, `List`) and `ErrNotFound` sentinel error
- [ ] 1.2 Create `internal/secret/keychain.go` with `KeychainStore` implementation using macOS `security` CLI and injectable command runner for testability
- [ ] 1.3 Create `internal/secret/keychain_test.go` with mocked command runner tests: set, get, delete, list, key not found, command failure
- [ ] 1.4 Create `internal/secret/mock_store.go` with testify `MockStore` for consumer tests
- [ ] 1.5 Verify build and all tests pass

## 2.0 Encrypted File Backend + Auto-Detection

- [ ] 2.1 Create `internal/secret/encrypted.go` with `EncryptedFileStore` implementation (AES-256-GCM + Argon2id, `~/.spinner/secrets.enc`, atomic writes, `0600` permissions)
- [ ] 2.2 Create `internal/secret/encrypted_test.go` with tests: round-trip set/get, multiple keys, delete, list, wrong passphrase error, corrupted file error, missing file returns empty store
- [ ] 2.3 Create `internal/secret/detect.go` with `NewStore()` auto-detection: darwin + `security` binary → Keychain, else → encrypted file, override via `SPINNER_SECRET_BACKEND` env var
- [ ] 2.4 Create `internal/secret/detect_test.go` with tests: override env var, platform-based selection
- [ ] 2.5 Verify build and all tests pass

## 3.0 `spinner secret` CLI Subcommand

- [ ] 3.1 Add `flagSecret` constant to `cmd/helpers.go`
- [ ] 3.2 Create `cmd/secret.go` with `spinner secret` parent command and `set`, `list`, `delete` subcommands using Store interface injection
- [ ] 3.3 Create `cmd/secret_test.go` with MockStore-based tests: set (prompted and `--value`), list, delete, delete nonexistent key error
- [ ] 3.4 Verify build and all tests pass

## 4.0 Secret Resolver + Spin Command Integration

- [ ] 4.1 Create `internal/secret/resolver.go` with `Resolve(store, customKeys)` function: store-first resolution for built-in tokens with env fallback, store-only for custom keys, error on missing required tokens
- [ ] 4.2 Create `internal/secret/resolver_test.go` with tests: token from store, token from env fallback, store overrides env, missing token error, custom key from store, custom key not found error
- [ ] 4.3 Add `Secrets map[string]string` to `provider.CreateConfig`
- [ ] 4.4 Modify `cmd/spin.go`: add `--secret` flag (StringSliceVar), create Store, call Resolve, populate `CreateConfig.Secrets`, remove `prerequisites.CheckEnvironmentVariables()` call
- [ ] 4.5 Modify `internal/backend/docker/run.go`: add `Secrets` to `spinConfig`, read `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from `config.Secrets` instead of `os.Getenv()`, write `--secret` values to env file
- [ ] 4.6 Modify `internal/backend/docker/docker_provider.go`: pass `Secrets` from `CreateConfig` to `spinConfig`
- [ ] 4.7 Modify `internal/backend/gcp/gcp_provider.go`: read `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from `config.Secrets` instead of `os.Getenv()`
- [ ] 4.8 Remove `prerequisites.CheckEnvironmentVariables()` function (replaced by resolver)
- [ ] 4.9 Update all affected tests: `cmd/spin_test.go`, `docker/run_test.go`, `docker/docker_provider_test.go`, `gcp/gcp_provider_test.go`, `prerequisites/prerequisites_test.go`
- [ ] 4.10 Verify build and all tests pass

## 5.0 Documentation

- [ ] 5.1 Update `docs/usage.md` with secret management workflow: `spinner secret set/list/delete`, `--secret` flag on spin, env var fallback behavior
