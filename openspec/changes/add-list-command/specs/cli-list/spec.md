# cli-list Specification

## Purpose

Provide a CLI command for discovering and displaying all spinner-managed instances across configured backends,
with rich execution state to help users identify running, stale, completed, or errored instances.

## ADDED Requirements

### Requirement: List Command

The CLI SHALL provide a `spinner list` command that discovers and displays all spinner-managed instances across
configured backends.

#### Scenario: List all instances

- **WHEN** user runs `spinner list`
- **THEN** the CLI SHALL query all registered backends for spinner-managed instances
- **AND** display a table with columns: BACKEND, NAME, STATUS, STATE, ITER, AGE, LAST UPDATE
- **AND** sort results by backend, then status (running first), then name

#### Scenario: No instances found

- **WHEN** user runs `spinner list` and no instances exist
- **THEN** the CLI SHALL print "No instances found"

#### Scenario: Backend unavailable

- **WHEN** a backend fails to initialize (e.g., Docker not running, GCP not configured)
- **THEN** the CLI SHALL print a warning for that backend
- **AND** continue listing instances from other available backends

#### Scenario: GCP backend auto-detection

- **WHEN** GCP project/zone configuration exists (from `--project`/`--zone` flags, `.spinner.json`, or env vars)
- **THEN** the CLI SHALL include GCP instances in the listing
- **AND** if `--state-bucket` is configured, include execution state from GCS

#### Scenario: GCP not configured

- **WHEN** no GCP project/zone configuration is available
- **THEN** the CLI SHALL silently skip the GCP backend (no warning)

### Requirement: Backend Filter

The `spinner list` command SHALL support a `--backend` flag to restrict listing to a single backend.

#### Scenario: Filter by backend

- **WHEN** user runs `spinner list --backend docker`
- **THEN** the CLI SHALL only query the Docker backend
- **AND** skip all other backends

#### Scenario: Invalid backend

- **WHEN** user provides an unregistered backend name
- **THEN** the CLI SHALL return an error indicating valid backends

### Requirement: JSON Output

The `spinner list` command SHALL support a `--json` flag for machine-readable output.

#### Scenario: JSON format

- **WHEN** user runs `spinner list --json`
- **THEN** the CLI SHALL output a JSON array of instance objects
- **AND** each object SHALL include: name, status, backend, image, repo, branch, agent, iteration,
  maxIterations, agentStatus, startedAt, lastUpdated

### Requirement: Rich State Display

The list output SHALL include execution state from state files alongside instance lifecycle status.

#### Scenario: Running instance with state

- **WHEN** an instance is running and has a state file
- **THEN** the output SHALL show the agent status (running/completed/rate_limited/error)
- **AND** the current iteration count and max iterations
- **AND** the time since the instance started (age)
- **AND** the time since the state was last updated

#### Scenario: Stale instance warning

- **WHEN** a running instance has not updated its state in more than 2 hours
- **THEN** the output SHALL display a stale warning indicator

#### Scenario: Instance without state

- **WHEN** an instance exists but has no state file (e.g., never ran or state was cleaned up)
- **THEN** the state-related columns SHALL show dashes or be empty
