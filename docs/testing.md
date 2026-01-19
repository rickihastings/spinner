# Testing Guidelines

## Test Coverage Requirements

- **All functionality must have tests**: Every new feature, bug fix, or significant code change must include appropriate
  tests
- **Test at the right level**: Use integration tests for end-to-end scenarios, unit tests for business logic
- **No untested code**: If code cannot be easily tested, it's a sign the design needs improvement

## Test Scripts

- Integration tests in `tests/` directory
- Each test file focuses on one scenario
- Tests are numbered for clarity (e.g., `11-prompt-without-branch.sh`)

## Testability Guidelines

- Write pure functions with clear inputs/outputs
- Avoid side effects where possible
- Use dependency injection for external dependencies
- Mock system calls in unit tests
