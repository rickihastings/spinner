## ADDED Requirements

### Requirement: Prerequisite Verification

The CLI SHALL verify that required tools are installed before proceeding with setup operations. The verification MUST
check for docker, git, and claude CLI tools. If any prerequisite is missing, the CLI SHALL exit immediately with an
error message identifying the missing tool.

#### Scenario: All prerequisites installed

- **WHEN** docker, git, and claude are all available in PATH
- **THEN** the CLI proceeds with the setup command

#### Scenario: Docker not installed

- **WHEN** docker is not available in PATH
- **THEN** the CLI exits with error code 1 and displays "Error: docker is not installed"

#### Scenario: Git not installed

- **WHEN** git is not available in PATH
- **THEN** the CLI exits with error code 1 and displays "Error: git is not installed"

#### Scenario: Claude not installed

- **WHEN** claude is not available in PATH
- **THEN** the CLI exits with error code 1 and displays "Error: claude is not installed"

### Requirement: Setup Command Flags

The setup command SHALL accept the following CLI flags: --name (required), --jvm-url (required), and --node-version (optional, defaults to 20). The CLI SHALL NOT prompt for interactive input.

#### Scenario: All required flags provided

- **WHEN** user runs `setup --name my-sandbox --jvm-url https://download.oracle.com/java/25/latest/jdk-25_linux-aarch64_bin.tar.gz`
- **THEN** the CLI proceeds with Docker image build

#### Scenario: Missing required flag

- **WHEN** user runs `setup` without --name or --jvm-url
- **THEN** the CLI exits with error code 1 and displays usage information

#### Scenario: Custom node version

- **WHEN** user runs `setup --name test --jvm-url https://download.oracle.com/java/25/latest/jdk-25_linux-aarch64_bin.tar.gz --node-version 20`
- **THEN** the Docker image is built with Node.js version 20

### Requirement: Docker Image Build

The CLI SHALL build a Docker image based on Ubuntu 22.04 containing JDK, nvm with Node.js, git, and
claude-code. The JDK SHALL be downloaded from the URL provided via the --jvm-url flag during build. The image SHALL be tagged as docker-sandbox:<name>.

#### Scenario: Successful image build

- **WHEN** setup command completes successfully
- **THEN** a Docker image named docker-sandbox:<name> exists locally

#### Scenario: JDK inclusion

- **WHEN** the Docker image is built with a valid --jvm-url
- **THEN** the container can execute `java --version` successfully

#### Scenario: Node.js inclusion

- **WHEN** the Docker image is built
- **THEN** the container can execute `node --version` and `npm --version` successfully

#### Scenario: Git inclusion

- **WHEN** the Docker image is built
- **THEN** the container can execute `git --version` successfully

#### Scenario: Claude-code inclusion

- **WHEN** the Docker image is built
- **THEN** the container can execute `claude --version` successfully

### Requirement: No Secrets in Image

The Docker image SHALL NOT contain any authentication tokens, API keys, or secrets. Secrets MUST be mounted at container
runtime.

#### Scenario: Clean image

- **WHEN** the Docker image is built
- **THEN** no environment variables containing tokens are baked into the image
- **AND** no credential files are present in the image filesystem

### Requirement: JVM Download

The Docker build process SHALL download the JDK from the URL provided via the --jvm-url flag. The user is responsible for providing a URL compatible with the target container architecture.

#### Scenario: Successful JDK download

- **WHEN** the --jvm-url points to a valid JDK tarball
- **THEN** the JDK is downloaded and installed in the Docker image

#### Scenario: Invalid JVM URL

- **WHEN** the --jvm-url points to an inaccessible or invalid resource
- **THEN** the Docker build fails with an appropriate error message
