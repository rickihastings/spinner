# Spinner

CLI tool for running code in isolated Docker containers.

## Features

- **Setup Command**: Build Docker sandbox images with JDK and Node.js
- **Spin Command**: Spin up development containers from pre-built images with repository cloning

## Installation

```bash
npm install
npm run build
npm link  # Optional: to use globally as 'spinner'
```

## Usage

### Setup Command

Build a Docker sandbox image with JDK and Node.js:

```bash
spinner setup --name my-sandbox --jvm-url <jdk-url> [--node-version 20]
```

Example:
```bash
spinner setup --name my-env --jvm-url https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.6%2B7/OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.6_7.tar.gz
```

### Spin Command

Spin up a development container with a cloned repository:

```bash
spinner spin --image <docker-image> --repo <git-ssh-url>
```

Example:
```bash
spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git
```

The container will:
- Mount your SSH agent for git authentication
- Mount your ~/.npmrc for npm registry access
- Clone the repository into /workspace
- Run in the background for multiple exec sessions

Access the container:
```bash
docker exec -it <container-name> bash
```

## Testing

Run the complete test suite:

```bash
./tests/run.sh
```

See [tests/README.md](tests/README.md) for more details.

## Requirements

- Docker
- Git
- Claude CLI
- Node.js
- SSH agent running (for spin command)