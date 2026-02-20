# Tasks: add-anthropic-api-key

## Slice 1.0 — Core resolver + tests + docs

- [ ] 1.1 Update `internal/secret/resolver.go`:
  - Replace `CLAUDE_CODE_OAUTH_TOKEN` in `builtInKeys` with an OR-group
    that tries `ANTHROPIC_API_KEY` first, then `CLAUDE_CODE_OAUTH_TOKEN`.
  - Produce a clear combined error if neither key is in the store.

- [ ] 1.2 Update `cmd/spin.go`:
  - Add `ANTHROPIC_API_KEY` to the `reserved` map.

- [ ] 1.3 Update `internal/secret/resolver_test.go`:
  - `TestResolve_AnthropicApiKeyPreferred` — store has both; assert
    `ANTHROPIC_API_KEY` wins.
  - `TestResolve_FallsBackToOAuthToken` — only `CLAUDE_CODE_OAUTH_TOKEN`
    in store; succeeds and map contains `CLAUDE_CODE_OAUTH_TOKEN`.
  - `TestResolve_AnthropicApiKeyOnly` — only `ANTHROPIC_API_KEY` in store;
    succeeds and map contains `ANTHROPIC_API_KEY`.
  - `TestResolve_NeitherClaudeAuthKey` — neither key in store; returns error
    with both key names in message.
  - Update existing `TestResolve_AllTokensFromStore` to reflect the new
    OR-group semantics.

- [ ] 1.4 No changes to `tests/testutil/secret.go`:
  - Integration tests keep seeding `CLAUDE_CODE_OAUTH_TOKEN` — they use a real
    OAuth token and adding `ANTHROPIC_API_KEY` would require a real API key in CI.
  - All new `ANTHROPIC_API_KEY` scenarios are covered by unit tests with mock
    store (task 1.3).

- [ ] 1.5 Update `docs/usage.md`:
  - Revise "Initial Setup" to present both keys as equal peers.
  - List `ANTHROPIC_API_KEY` first (since it is checked first at runtime).
  - Make clear only one is required.
  - Update the "Built-in tokens" description to name both options.

- [ ] 1.6 Verify: `go build ./...` passes; `go test ./...` passes.
- [ ] 1.7 Commit with message: `feat(secret): support ANTHROPIC_API_KEY as claude auth alternative`.
