# cli-spin Spec Delta

## ADDED Requirements

### Requirement: Unit Test Coverage for Spin Command Flags
The spin command SHALL have comprehensive unit tests that validate all flag combinations and validation logic without requiring Docker operations.

#### Scenario: Test missing image flag validation
- **GIVEN** spin command is invoked without --image flag
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "--image flag is required"

#### Scenario: Test missing repo flag validation
- **GIVEN** spin command is invoked without --repo flag
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "--repo flag is required"

#### Scenario: Test prompt flag parsing
- **GIVEN** spin command is invoked with --prompt flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the prompt value is correctly extracted

#### Scenario: Test branch flag parsing
- **GIVEN** spin command is invoked with --branch flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the branch value is correctly extracted

#### Scenario: Test max-iterations flag parsing
- **GIVEN** spin command is invoked with --max-iterations flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the max-iterations value is correctly extracted

#### Scenario: Test recreate flag parsing
- **GIVEN** spin command is invoked with --recreate flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the recreate boolean is correctly set

#### Scenario: Test max-iterations default value
- **GIVEN** spin command is invoked without --max-iterations flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify max-iterations defaults to 30

### Requirement: Unit Test Coverage for Container Operations
The Docker container operations SHALL have unit tests with mocked Docker client to verify run logic without actual Docker calls.

#### Scenario: Test container creation with mocked Docker
- **GIVEN** a mocked Docker client that returns success
- **WHEN** RunContainer is called with valid configuration
- **THEN** the test SHALL verify the container is created without error

#### Scenario: Test container naming logic
- **GIVEN** image, repo, and branch parameters
- **WHEN** container name is generated
- **THEN** the test SHALL verify the name follows the deterministic format

#### Scenario: Test container name sanitization
- **GIVEN** repo URL with special characters
- **WHEN** container name is generated
- **THEN** the test SHALL verify special characters are replaced with hyphens

#### Scenario: Test container reuse logic
- **GIVEN** a container with the same name already exists
- **WHEN** spin command is executed without --recreate flag
- **THEN** the test SHALL verify the existing container is reused

#### Scenario: Test container recreation logic
- **GIVEN** a container with the same name already exists
- **WHEN** spin command is executed with --recreate flag
- **THEN** the test SHALL verify the existing container is removed and recreated

### Requirement: Unit Test Coverage for Prerequisites
The prerequisite validation SHALL have unit tests that verify all required tokens and tools are checked properly.

#### Scenario: Test GitHub token validation
- **GIVEN** GITHUB_TOKEN environment variable is not set
- **WHEN** prerequisites are checked
- **THEN** the test SHALL verify an error is returned indicating missing GitHub token

#### Scenario: Test Claude token validation
- **GIVEN** ANTHROPIC_API_KEY environment variable is not set
- **WHEN** prerequisites are checked
- **THEN** the test SHALL verify an error is returned indicating missing Anthropic API key

#### Scenario: Test Docker availability check
- **GIVEN** Docker is not available on the system
- **WHEN** prerequisites are checked
- **THEN** the test SHALL verify an error is returned indicating Docker is required

### Requirement: Integration Test Coverage for Spin Command
The spin command SHALL have integration tests that verify end-to-end behavior with real Docker operations.

#### Scenario: Integration test for successful container creation
- **GIVEN** Docker is running and valid image exists
- **WHEN** spin command is executed with valid arguments
- **THEN** the test SHALL verify a Docker container is created and running

#### Scenario: Integration test for container naming
- **GIVEN** spin command is executed with image, repo, and branch
- **WHEN** docker ps is queried
- **THEN** the test SHALL verify container name matches expected format

#### Scenario: Integration test for repository cloning
- **GIVEN** spin command is executed with valid repo URL
- **WHEN** container filesystem is inspected
- **THEN** the test SHALL verify repository files exist in /workspace

#### Scenario: Integration test for branch checkout
- **GIVEN** spin command is executed with --branch flag
- **WHEN** container git status is checked
- **THEN** the test SHALL verify the specified branch is checked out

#### Scenario: Integration test for prompt execution
- **GIVEN** spin command is executed with --prompt flag
- **WHEN** container processes are inspected
- **THEN** the test SHALL verify ralph-loop is running with the prompt

#### Scenario: Integration test for container reuse
- **GIVEN** a container from previous spin command exists
- **WHEN** spin command is executed again without --recreate
- **THEN** the test SHALL verify the same container is reused

#### Scenario: Integration test for container recreation
- **GIVEN** a container from previous spin command exists
- **WHEN** spin command is executed with --recreate flag
- **THEN** the test SHALL verify old container is removed and new one is created

#### Scenario: Integration test for non-existent image
- **GIVEN** spin command is executed with non-existent image name
- **WHEN** Docker operations are attempted
- **THEN** the test SHALL verify appropriate error message is returned

#### Scenario: Integration test for max-iterations parameter
- **GIVEN** spin command is executed with --max-iterations flag
- **WHEN** container environment is inspected
- **THEN** the test SHALL verify MAX_ITERATIONS environment variable is set correctly

#### Scenario: Integration test cleanup
- **GIVEN** an integration test has created Docker containers
- **WHEN** the test completes (success or failure)
- **THEN** the test SHALL clean up created containers to prevent resource leaks

### Requirement: Table-Driven Test Coverage
Complex validation scenarios SHALL use table-driven tests to comprehensively cover all edge cases and variations.

#### Scenario: Table-driven tests for container naming
- **GIVEN** various combinations of repo URLs and branch names
- **WHEN** container names are generated
- **THEN** the test SHALL verify correct sanitization and formatting for all cases

#### Scenario: Table-driven tests for flag combinations
- **GIVEN** various combinations of flags (prompt, branch, max-iterations, recreate)
- **WHEN** spin command is executed
- **THEN** the test SHALL verify correct behavior for all valid combinations

## MODIFIED Requirements

None - this change adds testing requirements without modifying existing functional requirements.

## REMOVED Requirements

None - bash tests will be deprecated but not removed initially for validation purposes.
