# cli-setup Spec Delta

## ADDED Requirements

### Requirement: Unit Test Coverage for Setup Command
The setup command SHALL have comprehensive unit tests that validate argument parsing, flag validation, and error handling without requiring Docker operations.

#### Scenario: Test missing name flag validation
- **GIVEN** setup command is invoked without --name flag
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "Missing required flag"

#### Scenario: Test mutually exclusive flags validation
- **GIVEN** setup command is invoked with both --base-image and --dockerfile flags
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "mutually exclusive"

#### Scenario: Test successful argument parsing
- **GIVEN** setup command is invoked with valid --name flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the name value is correctly extracted

#### Scenario: Test base-image flag parsing
- **GIVEN** setup command is invoked with --name and --base-image flags
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify both flag values are correctly extracted

#### Scenario: Test dockerfile flag parsing
- **GIVEN** setup command is invoked with --name and --dockerfile flags
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify both flag values are correctly extracted

#### Scenario: Test dockerfile path validation
- **GIVEN** setup command is invoked with non-existent dockerfile path
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned indicating file not found

### Requirement: Unit Test Coverage for Docker Build Logic
The Docker build operations SHALL have unit tests with mocked Docker client to verify build logic without actual Docker calls.

#### Scenario: Test successful image build with mocked Docker
- **GIVEN** a mocked Docker client that returns success
- **WHEN** BuildImage is called with valid configuration
- **THEN** the test SHALL verify the build completes without error

#### Scenario: Test image build with custom base image
- **GIVEN** a mocked Docker client
- **WHEN** BuildImage is called with custom base-image configuration
- **THEN** the test SHALL verify the correct base image is used in Docker build

#### Scenario: Test image build with custom Dockerfile
- **GIVEN** a mocked Docker client
- **WHEN** BuildImage is called with dockerfile configuration
- **THEN** the test SHALL verify the custom Dockerfile is used in Docker build

#### Scenario: Test image build failure handling
- **GIVEN** a mocked Docker client that returns build error
- **WHEN** BuildImage is called
- **THEN** the test SHALL verify the error is properly propagated

### Requirement: Integration Test Coverage for Setup Command
The setup command SHALL have integration tests that verify end-to-end behavior with real Docker operations.

#### Scenario: Integration test for successful image build
- **GIVEN** Docker is running on the host system
- **WHEN** setup command is executed with valid arguments
- **THEN** the test SHALL verify a Docker image is created with the specified name

#### Scenario: Integration test for image existence verification
- **GIVEN** setup command has successfully built an image
- **WHEN** docker images is queried
- **THEN** the test SHALL verify the image exists in Docker's image list

#### Scenario: Integration test for git installation in image
- **GIVEN** setup command has successfully built an image
- **WHEN** a container is run from the image to check git
- **THEN** the test SHALL verify git is available in the image

#### Scenario: Integration test for claude-code installation in image
- **GIVEN** setup command has successfully built an image
- **WHEN** a container is run from the image to check claude-code
- **THEN** the test SHALL verify claude-code is available in the image

#### Scenario: Integration test cleanup
- **GIVEN** an integration test has created Docker images
- **WHEN** the test completes (success or failure)
- **THEN** the test SHALL clean up created images to prevent resource leaks

### Requirement: Test Utility Infrastructure
The project SHALL provide test utilities for Docker operations, CLI execution, and resource cleanup to support both unit and integration tests.

#### Scenario: Docker test helpers available
- **GIVEN** tests need to verify Docker resource state
- **WHEN** test utilities are imported
- **THEN** helper functions for image existence, container status, and cleanup SHALL be available

#### Scenario: CLI execution helpers available
- **GIVEN** integration tests need to execute CLI commands
- **WHEN** test utilities are imported
- **THEN** helper functions for building CLI binary and running commands SHALL be available

#### Scenario: Test cleanup utilities available
- **GIVEN** tests create Docker resources
- **WHEN** tests complete or fail
- **THEN** cleanup utilities SHALL ensure resources are removed

## MODIFIED Requirements

None - this change adds testing requirements without modifying existing functional requirements.

## REMOVED Requirements

None - bash tests will be deprecated but not removed initially for validation purposes.
