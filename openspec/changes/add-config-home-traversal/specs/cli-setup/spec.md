## MODIFIED Requirements

### Requirement: Configuration File Support

The setup command SHALL read infrastructure defaults from a `.spinner.json` file discovered by searching from the current
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

#### Scenario: Config file provides backend default

- **WHEN** `.spinner.json` contains `{"backend": "gcp", "project": "my-proj", "zone": "us-central1-a", "state-bucket": "my-bucket"}`
- **AND** user runs `spinner setup --name my-env`
- **THEN** the CLI SHALL use the GCP backend with values from the config file

#### Scenario: CLI flags override config file

- **WHEN** `.spinner.json` contains `{"zone": "us-central1-a"}`
- **AND** user runs `spinner setup --backend gcp --name my-env --zone us-east1-b`
- **THEN** the CLI SHALL use `us-east1-b` (CLI flag takes precedence)

#### Scenario: Environment variables override config file

- **WHEN** `.spinner.json` contains `{"project": "file-project"}`
- **AND** `SPINNER_PROJECT=env-project` is set
- **THEN** the CLI SHALL use `env-project` (env var takes precedence over config file)

#### Scenario: No config file present

- **WHEN** no `.spinner.json` exists in the current directory, any ancestor directory, or `$HOME`
- **THEN** the CLI SHALL continue normally using CLI flags, env vars, and defaults

#### Scenario: Invalid config file

- **WHEN** `.spinner.json` exists but contains invalid JSON
- **THEN** the CLI SHALL print a warning and continue using CLI flags and defaults
