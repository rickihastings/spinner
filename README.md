# Spinner

CLI tool for running code in isolated Docker containers.

## Features

- **Setup Command**: Build Docker sandbox images with custom base images or Dockerfiles
- **Spin Command**: Spin up development containers from pre-built images with repository cloning

## Installation

### Prerequisites

- Docker
- Git
- Go 1.21+ (for building from source)
- GitHub Personal Access Token (for spin command)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/rickihastings/spinner.git
cd spinner

# Build the binary
go build -o dist/spinner

# Optional: Install globally
cp dist/spinner /usr/local/bin/spinner
# Or add dist/ to your PATH
```

### Quick Install

```bash
go build -o dist/spinner
# Then use ./dist/spinner or copy to your PATH
```

## Usage

### Setup Command

Build a Docker sandbox image with a custom base image or Dockerfile:

```bash
spinner setup --name my-sandbox [--base-image <image> | --dockerfile <path>]
```

The setup command ensures git and claude-code are installed in the final image. You can:

- Use the default ubuntu:22.04 base (no flags needed)
- Specify a custom base image with `--base-image`
- Provide your own Dockerfile with `--dockerfile`

**Note**: Only Ubuntu/Debian-based images are supported (requires apt-get).

Examples:

```bash
# Use default ubuntu:22.04 base
spinner setup --name my-env

# Use a Node.js base image
spinner setup --name node-env --base-image node:20-bullseye

# Use a custom Dockerfile
spinner setup --name custom-env --dockerfile ./Dockerfile.custom
```

### Spin Command

Spin up a development container with a cloned repository:

```bash
export GITHUB_TOKEN=<your-github-token>
spinner spin --image <docker-image> --repo <git-url>
```

Example:

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
spinner spin --image spinner:my-env --repo https://github.com/octocat/Hello-World.git
```

The container will:

- Use GitHub Personal Access Token for git authentication
- Mount your ~/.npmrc for npm registry access
- Clone the repository into /home/spinner/workspace
- Run in the background for multiple exec sessions

Access the container:

```bash
docker exec -it <container-name> bash
```

#### GitHub Token Setup

To use the spin command, you need a GitHub Personal Access Token:

1. Generate a token at https://github.com/settings/tokens
2. Required scopes:
    - `repo` - Full control of private repositories (required for private repos)
    - For public repos only, no scopes are technically required, but `public_repo` is recommended
3. Set the token as an environment variable:
   ```bash
   export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
   ```

**Security Note**: The token is passed to the container via environment variable (not CLI flag) to prevent exposure in
bash history.

## Development

### Building

```bash
# Build the binary
go build -o dist/spinner

# Build with specific flags
go build -ldflags "-s -w" -o dist/spinner  # Smaller binary
```

### Testing

The project uses Go's native testing framework with both unit and integration tests.

```bash
# Run all tests (unit + integration)
go test ./...

# Run only unit tests (fast, no Docker required)
go test -short ./...

# Run only integration tests (requires Docker)
go test ./tests/integration/...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

**Prerequisites for integration tests:**
- Docker installed and running
- `GITHUB_TOKEN` environment variable set
- `CLAUDE_CODE_OAUTH_TOKEN` environment variable set

See [tests/README.md](tests/README.md) for comprehensive testing documentation.

## Project Structure

```
.
├── cmd/                    # Command implementations (Cobra commands)
│   ├── root.go            # Root command with version and help
│   ├── setup.go           # Setup command implementation
│   └── spin.go            # Spin command implementation
├── internal/              # Internal packages (not importable by external projects)
│   ├── docker/            # Docker operations and Dockerfile generation
│   └── prerequisites/     # Prerequisite checking logic
├── dist/                  # Build output directory
│   └── spinner           # Compiled binary
├── tests/                 # Integration tests
├── docs/                  # Documentation
└── main.go               # Entry point

```

## Requirements

- Docker
- Git
- Go 1.21+ (for building)
- GitHub Personal Access Token (for spin command)