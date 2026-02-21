## ADDED Requirements

### Requirement: Claude auth secret resolution SHALL use ANTHROPIC_API_KEY with fallback to CLAUDE_CODE_OAUTH_TOKEN

When resolving built-in secrets before launching a spin, the resolver MUST check
`ANTHROPIC_API_KEY` first. If that key is present in the store, it SHALL be used and
`CLAUDE_CODE_OAUTH_TOKEN` MUST NOT be queried. If `ANTHROPIC_API_KEY` is absent, the
resolver SHALL fall back to `CLAUDE_CODE_OAUTH_TOKEN`. At least one of the two MUST
be present; if neither is found the command MUST fail with an error naming both keys.

#### Scenario: ANTHROPIC_API_KEY takes priority when both are stored

Given the secret store contains both `ANTHROPIC_API_KEY` and `CLAUDE_CODE_OAUTH_TOKEN`
When `spinner spin` resolves built-in secrets
Then the resolved map contains `ANTHROPIC_API_KEY` and its value
And the resolved map does NOT contain `CLAUDE_CODE_OAUTH_TOKEN`

#### Scenario: Falls back to CLAUDE_CODE_OAUTH_TOKEN when only it is stored

Given the secret store contains `CLAUDE_CODE_OAUTH_TOKEN` but not `ANTHROPIC_API_KEY`
When `spinner spin` resolves built-in secrets
Then the resolved map contains `CLAUDE_CODE_OAUTH_TOKEN` and its value
And no error is returned

#### Scenario: Succeeds when only ANTHROPIC_API_KEY is stored

Given the secret store contains `ANTHROPIC_API_KEY` but not `CLAUDE_CODE_OAUTH_TOKEN`
When `spinner spin` resolves built-in secrets
Then the resolved map contains `ANTHROPIC_API_KEY` and its value
And no error is returned

#### Scenario: Fails with helpful error when neither Claude auth key is stored

Given the secret store contains neither `ANTHROPIC_API_KEY` nor `CLAUDE_CODE_OAUTH_TOKEN`
When `spinner spin` resolves built-in secrets
Then an error is returned
And the error message mentions both `ANTHROPIC_API_KEY` and `CLAUDE_CODE_OAUTH_TOKEN`
And the error message includes instructions to run `spinner secret set`

### Requirement: ANTHROPIC_API_KEY MUST be a reserved variable name

`ANTHROPIC_API_KEY` SHALL be reserved and MUST NOT be supplied via `--secret` or any
user-provided environment variable mechanism.

#### Scenario: User attempts to pass ANTHROPIC_API_KEY via --secret flag

Given a user runs `spinner spin --secret ANTHROPIC_API_KEY ...`
Then the command returns an error indicating the key is reserved
And the container is not started
