## MODIFIED Requirements

### Requirement: Configuration File Support

The spin command SHALL read infrastructure defaults from a `.spinner.json` file discovered by searching from the current
working directory upward through ancestor directories, with a fallback to `$HOME/.spinner.json`.

#### Scenario: Config file in current directory

- **WHEN** `.spinner.json` exists in the current working directory
- **THEN** the CLI SHALL load that file as the configuration source

#### Scenario: Config file in ancestor directory

- **WHEN** no `.spinner.json` exists in the current working directory
- **AND** a `.spinner.json` exists in an ancestor directory (e.g., `$HOME/.spinner.json` when cwd is `$HOME/projects/repo`)
- **THEN** the CLI SHALL traverse upward from cwd and load the nearest `.spinner.json` found

#### Scenario: Config file in home directory as fallback

- **WHEN** no `.spinner.json` exists in the current directory or any ancestor directory
- **AND** `$HOME/.spinner.json` exists
- **THEN** the CLI SHALL load `$HOME/.spinner.json` as a fallback

#### Scenario: First config file wins (no merging)

- **WHEN** `.spinner.json` exists in both the current directory and `$HOME`
- **THEN** the CLI SHALL load only the nearest file (current directory)
- **AND** the home directory file SHALL be ignored entirely (no merging)

#### Scenario: Config file provides full GCP config

- **WHEN** `.spinner.json` contains `{"backend": "gcp", "project": "p", "zone": "z", "state-bucket": "b"}`
- **AND** user runs `spinner spin --image my-env --repo <url> --prompt "Fix bug"`
- **THEN** the CLI SHALL use the GCP backend with project, zone, and state-bucket from the config file

#### Scenario: CLI flags override config file

- **WHEN** `.spinner.json` contains `{"machine-type": "e2-standard-2"}`
- **AND** user runs `spinner spin --backend gcp --image my-env --repo <url> --machine-type n2-standard-4`
- **THEN** the CLI SHALL use `n2-standard-4` (CLI flag takes precedence)

#### Scenario: No config file present

- **WHEN** no `.spinner.json` exists in the current directory, any ancestor directory, or `$HOME`
- **THEN** the CLI SHALL continue normally using CLI flags, env vars, and defaults
