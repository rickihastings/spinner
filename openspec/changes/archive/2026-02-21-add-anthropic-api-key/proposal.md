# Proposal: add-anthropic-api-key

## Summary

Add `ANTHROPIC_API_KEY` as an alternative authentication secret alongside
`CLAUDE_CODE_OAUTH_TOKEN`. When resolving Claude auth at spin time, the secret
resolver checks `ANTHROPIC_API_KEY` first; if absent it falls back to
`CLAUDE_CODE_OAUTH_TOKEN`. Exactly one of the two must be present in the
secret store — neither is individually mandatory, but at least one is required.

## Motivation

`CLAUDE_CODE_OAUTH_TOKEN` requires users to have gone through the Claude Code
OAuth flow. `ANTHROPIC_API_KEY` is the standard Anthropic API credential and
is simpler to obtain for users who already have an Anthropic account. Supporting
both widens Spinner's addressable audience and removes a friction point for
new users.

## Scope

- **`internal/secret/resolver.go`** — replace the hard-coded `CLAUDE_CODE_OAUTH_TOKEN`
  built-in with an OR-group that tries `ANTHROPIC_API_KEY` first, then falls
  back to `CLAUDE_CODE_OAUTH_TOKEN`. The resolved secret is stored in the map
  under whichever key was actually found, so the correct env-var name reaches
  the Claude CLI process.

- **`cmd/spin.go`** — add `ANTHROPIC_API_KEY` to the `reserved` map so it
  cannot be overridden via `--secret`.

- **`docs/usage.md`** — update the "Initial Setup" section to show
  `ANTHROPIC_API_KEY` as the recommended first choice, with
  `CLAUDE_CODE_OAUTH_TOKEN` listed as the alternative.

- **Tests** — update `internal/secret/resolver_test.go` and
  `tests/testutil/secret.go` to cover ANTHROPIC_API_KEY-first, fallback, and
  neither-present scenarios.

## Design Decisions

### Priority and fallback
`ANTHROPIC_API_KEY` is checked first; if present it is used and
`CLAUDE_CODE_OAUTH_TOKEN` is not queried. This avoids ambiguity about which key
the Claude CLI process receives when both are stored.

### Both keys present
If both are in the store, `ANTHROPIC_API_KEY` wins silently. No warning is
emitted — having both is a fully valid state.

### Docs framing
Both keys are presented as equal peers. The docs show `ANTHROPIC_API_KEY` first
(since it is checked first at runtime) but make clear either works and only one
is required. No deprecation hint for `CLAUDE_CODE_OAUTH_TOKEN`.

### Error message when neither is present
```
claude auth secret not found — set one with:
  spinner secret set ANTHROPIC_API_KEY
or:
  spinner secret set CLAUDE_CODE_OAUTH_TOKEN
```

### Reserved-variable protection
`ANTHROPIC_API_KEY` is added to the `reserved` map in `cmd/spin.go` alongside
`CLAUDE_CODE_OAUTH_TOKEN` so users cannot supply it via `--secret` and
accidentally override the resolved value.

### No changes to the container / exec layer
The secrets blob already carries whatever key→value pairs are resolved.
Because we store the secret under its actual key name (`ANTHROPIC_API_KEY` or
`CLAUDE_CODE_OAUTH_TOKEN`), the Claude CLI process receives the right env var
automatically — no changes needed in `internal/exec/loop.go` or the GCP
runtime script.

### Test strategy
- **Integration tests** (`tests/testutil/secret.go`): keep seeding
  `CLAUDE_CODE_OAUTH_TOKEN` — integration tests use a real OAuth token and
  adding `ANTHROPIC_API_KEY` would require a real API key in CI.
- **Unit tests** (`internal/secret/resolver_test.go`): cover all four scenarios
  (API key only, OAuth only, both, neither) with mock store — no real credential
  needed.
