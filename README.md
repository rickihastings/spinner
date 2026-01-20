# Spinner

CLI tool for running code in isolated Docker containers.

## Features

- **Setup Command**: Build Docker sandbox images with custom base images or Dockerfiles
- **Spin Command**: Spin up development containers from pre-built images with repository cloning

## Installation

```bash
npm install
npm run build
npm link  # Optional: to use globally as 'spinner'
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
- GitHub Personal Access Token (for spin command)