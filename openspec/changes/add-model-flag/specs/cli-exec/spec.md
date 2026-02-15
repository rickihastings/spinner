# cli-exec Delta: add-model-flag

## MODIFIED Requirements

### Requirement: Configuration Loading

The exec command SHALL load configuration from environment variables.

#### Scenario: All environment variables present

- **WHEN** PROMPT, MAX_ITERATIONS, LOG_DIR, BRANCH, STATE_DIR are set
- **THEN** configuration SHALL be loaded with all values

#### Scenario: Optional environment variables missing

- **WHEN** PROMPT and MAX_ITERATIONS are set but LOG_DIR, STATE_DIR, and ANTHROPIC_MODEL are not
- **THEN** configuration SHALL use default STATE_DIR value of "/state" and empty model (Claude CLI default)

#### Scenario: Required environment variable missing

- **WHEN** PROMPT is not set
- **THEN** configuration loading SHALL fail with error "PROMPT environment variable is not set"

#### Scenario: ANTHROPIC_MODEL environment variable present

- **WHEN** ANTHROPIC_MODEL is set in the environment
- **THEN** configuration SHALL include the model value
- **AND** the Claude CLI SHALL use the specified model (via environment variable, no CLI arg needed)
