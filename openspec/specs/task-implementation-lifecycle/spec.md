# task-implementation-lifecycle Specification

## Purpose
TBD - created by archiving change add-task-implementation-lifecycle. Update Purpose after archive.
## Requirements
### Requirement: Task List Prerequisite

The skill SHALL require a task list to exist before beginning implementation. If no task list is found, the skill MUST
alert the user and halt execution.

#### Scenario: Task list not found

- **WHEN** the skill is invoked
- **AND** no task list exists in the expected location
- **THEN** the skill alerts the user that a task list is required
- **AND** execution halts immediately

#### Scenario: Task list found

- **WHEN** the skill is invoked
- **AND** a task list exists
- **THEN** the skill proceeds with the implementation lifecycle

### Requirement: Ten-Step Deterministic Lifecycle

The skill SHALL execute a deterministic 10-step lifecycle for each task implementation: (1) Orient, (2) Read Plan, (3)
Select Task, (4) Investigate, (5) Implement, (6) Validate, (7) Update Plan, (8) Update Spec on Divergence, (9) Commit, 
(10) Halt.

#### Scenario: Complete lifecycle execution

- **WHEN** a task is selected for implementation
- **THEN** the skill executes all 10 steps in order
- **AND** does not skip or reorder steps
- **AND** halts after step 10

### Requirement: Orientation Phase

The skill SHALL begin by understanding the goal and studying relevant specifications. This phase is mandatory regardless
of spec-driven development approach (OpenSpec, speckit, or other).

#### Scenario: Spec-agnostic orientation

- **WHEN** the orientation phase begins
- **THEN** the skill identifies and reads available specifications
- **AND** understands the implementation goal
- **AND** supports any spec format or methodology

### Requirement: Task Selection

The skill SHALL select the next incomplete task from the task list and mark it as in-progress before beginning
implementation work.

#### Scenario: Next task selected

- **WHEN** the task list contains incomplete tasks
- **THEN** the skill selects the next incomplete task
- **AND** marks it as in-progress
- **AND** proceeds to investigation

### Requirement: Non-Assumption Investigation

The skill SHALL investigate the codebase using subagents before implementing changes. The skill MUST NOT assume
functionality is not already implemented.

#### Scenario: Investigation before implementation

- **WHEN** a task is selected
- **THEN** the skill spawns subagents to explore the codebase
- **AND** searches for existing implementations of the required functionality
- **AND** verifies current state before making changes

### Requirement: Implementation Execution

The skill SHALL spawn one or more subagents to perform file operations during the implementation phase.

#### Scenario: File operations via subagents

- **WHEN** implementation begins
- **THEN** the skill uses subagents for file read, write, and edit operations
- **AND** maintains separation of concerns through agent delegation

### Requirement: Validation Phase

The skill SHALL validate the implementation by spawning a subagent to run builds and tests.

#### Scenario: Build and test validation

- **WHEN** implementation is complete
- **THEN** the skill spawns a validation subagent
- **AND** executes build processes
- **AND** runs available tests
- **AND** reports validation results

### Requirement: Task Completion Tracking

The skill SHALL update the implementation plan by marking the current task as complete after successful validation.

#### Scenario: Task marked complete

- **WHEN** validation passes
- **THEN** the skill updates the task list
- **AND** marks the current task as done
- **AND** preserves remaining incomplete tasks

### Requirement: Spec Divergence Detection and Update

The skill SHALL detect when implementation diverges from specification and instruct the agent to update the spec
accordingly. The skill MUST remain agnostic to the specific spec format.

#### Scenario: Divergence detected

- **WHEN** implementation differs from documented specification
- **THEN** the skill identifies the divergence
- **AND** instructs the agent to update the relevant spec document
- **AND** handles any spec format (OpenSpec, speckit, or other)

#### Scenario: No divergence

- **WHEN** implementation matches specification
- **THEN** the skill skips spec updates
- **AND** proceeds to commit phase

### Requirement: Commit After Task

The skill SHALL create a git commit after completing a task and updating documentation.

#### Scenario: Commit created

- **WHEN** a task is marked complete
- **AND** specs are updated if needed
- **THEN** the skill creates a git commit
- **AND** includes meaningful commit message describing changes

### Requirement: Mandatory Halt

The skill SHALL halt execution after completing one task and creating a commit. The skill MUST NOT continue to the next
task automatically.

#### Scenario: Halt after single task

- **WHEN** a task is completed and committed
- **THEN** the skill halts execution
- **AND** does not select the next task
- **AND** requires explicit re-invocation to continue

#### Scenario: Prevent runaway execution

- **WHEN** multiple incomplete tasks remain in the task list
- **THEN** the skill completes only the current task
- **AND** halts before processing additional tasks
- **AND** awaits user instruction to continue

### Requirement: Feature Completion Signal (IMPORTANT)

The skill SHALL output a distinctive completion signal when all tasks in the task list are marked complete, indicating the
entire feature implementation is finished. The signal MUST be "~~ FEATURE_COMPLETED ~~".

#### Scenario: All tasks completed

- **WHEN** the current task is completed and committed
- **AND** no incomplete tasks remain in the task list
- **THEN** the skill outputs the signal "~~ FEATURE_COMPLETED ~~"
- **AND** halts execution

#### Scenario: Tasks remain incomplete

- **WHEN** the current task is completed and committed
- **AND** incomplete tasks still exist in the task list
- **THEN** the skill does not output the completion signal
- **AND** halts normally awaiting re-invocation

### Requirement: Container Template Deployment

The skill SHALL be packaged as a template and copied to a Docker container image for isolated execution.

#### Scenario: Template in container

- **WHEN** the Docker image is built
- **THEN** the skill template is copied into the image
- **AND** is available for invocation within the container
- **AND** can execute in the sandboxed environment

