## MODIFIED Requirements

### Requirement: Setup documentation MUST present ANTHROPIC_API_KEY and CLAUDE_CODE_OAUTH_TOKEN as equal peers

User-facing documentation for initial setup SHALL list `ANTHROPIC_API_KEY` first
(reflecting runtime check order) and `CLAUDE_CODE_OAUTH_TOKEN` as the alternative.
The docs MUST make clear that both are valid and only one is required. Neither key
SHALL be described as deprecated or preferred.

#### Scenario: User follows setup docs using ANTHROPIC_API_KEY

Given a user reads the "Initial Setup" section of the usage docs
When they follow the primary instructions
Then they run `spinner secret set ANTHROPIC_API_KEY` (not the OAuth token)
And the subsequent `spinner spin` command succeeds with that key

#### Scenario: User follows setup docs using CLAUDE_CODE_OAUTH_TOKEN

Given a user reads the "Initial Setup" section of the usage docs
When they follow the alternative instructions
Then they run `spinner secret set CLAUDE_CODE_OAUTH_TOKEN`
And the subsequent `spinner spin` command succeeds with that key
