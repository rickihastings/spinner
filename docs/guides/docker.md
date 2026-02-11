# Docker Sandbox Guide

This guide walks you through using Spinner to run Claude agents in sandboxed Docker containers. You can either create a sandbox to work in interactively, or launch an autonomous agent loop that works on a task unsupervised.

## Prerequisites

Before you begin, make sure you have:

- **Docker** installed and running (`docker ps` should work without errors)
- **A GitHub token** exported as `GITHUB_TOKEN`
- **A Claude Code OAuth token** exported as `CLAUDE_CODE_OAUTH_TOKEN`
- **SSH agent** running on your host system

```bash
export GITHUB_TOKEN="ghp_your_token_here"
export CLAUDE_CODE_OAUTH_TOKEN="your_oauth_token_here"
```

## Step 1: Build a Sandbox Image

Before spinning up containers, you need to build a Docker image. This image comes pre-configured with git and Claude Code.

```bash
spinner setup --name my-sandbox
```

This uses `ubuntu:22.04` as the base. To use a different base image (must be Debian/Ubuntu-based):

```bash
spinner setup --name node-env --base-image node:20-bullseye
```

Or use your own Dockerfile:

```bash
spinner setup --name custom-env --dockerfile ./Dockerfile.custom
```

> **Note:** `--base-image` and `--dockerfile` cannot be used together.

### Customizing Claude Code Configuration

If you use custom MCP servers, slash commands, skills, or other Claude Code settings, you can bake them into the Docker
image so the agent has access at runtime. Create a Dockerfile that copies your `.claude/` directory into the spinner
user's home:

```dockerfile
FROM ubuntu:22.04

# Copy your Claude Code configuration.
# The spinner user (UID 1000) is created by the extending template that
# wraps this Dockerfile, so set ownership to UID/GID 1000.
COPY .claude/ /home/spinner/.claude/
RUN chown -R 1000:1000 /home/spinner/.claude/
```

Then build with:

```bash
spinner setup --name my-custom-env --dockerfile ./Dockerfile.custom
```

Spinner builds your Dockerfile first, then layers its own setup (git, Claude Code, the spinner binary) on top. Files
you placed in `/home/spinner/.claude/` are preserved because the extending template does not overwrite existing
directories.

**What to include:**

| Path                                    | Purpose                          |
|-----------------------------------------|----------------------------------|
| `.claude/settings.json`                 | MCP servers, permissions, prefs  |
| `.claude/commands/`                     | Custom slash commands             |
| `.claude/skills/`                       | Custom skills                     |

**Example with MCP servers and custom commands:**

```dockerfile
FROM ubuntu:22.04

# Install additional dependencies your MCP servers might need
RUN apt-get update && apt-get install -y python3 python3-pip && rm -rf /var/lib/apt/lists/*
RUN pip3 install my-mcp-server

# Copy Claude Code config from your local .claude directory
COPY .claude/settings.json /home/spinner/.claude/settings.json
COPY .claude/commands/ /home/spinner/.claude/commands/
RUN chown -R 1000:1000 /home/spinner/.claude/
```

> **Tip:** Only copy the specific files you need rather than your entire `~/.claude/` directory, which may contain
> tokens or session data that should not be baked into an image.

## Step 2: Spin Up a Container

Once your image is built, you spin up a container pointed at a git repository. How the container behaves depends on whether you provide a `--prompt`.

### Interactive Mode (No Prompt)

To create a sandbox you can exec into and use manually:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --branch main
```

This starts a container with your repository cloned and checked out. The container stays running, and you can exec into it to run Claude yourself:

```bash
docker exec -it <container-name> bash
```

From inside the container, you have a full development environment with git configured and Claude Code available. Run Claude interactively, install dependencies, or do whatever you need.

> **Tip:** The container name is deterministic based on your image and repo. For example, with `--image my-sandbox` and a repo named `your-repo`, the container name will be something like `spinner-my-sandbox-your-repo`. If you also pass `--branch main`, it becomes `spinner-my-sandbox-your-repo-main`.

### Autonomous Mode (With Prompt)

To kick off an autonomous agent loop that works on a task without supervision:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --branch main \
  --prompt "Refactor the authentication module to use JWT tokens"
```

When a prompt is provided, Spinner launches an iteration loop inside the container. Each iteration:

1. Runs Claude with your prompt
2. Checks the output for completion, errors, or rate limits
3. Pushes any changes to git
4. Saves progress to a state file
5. Repeats until the task is done or the iteration limit is reached

The agent signals completion by outputting `~~ FEATURE_COMPLETED ~~`. At that point, the loop stops and your changes are on the branch.

By default, the loop runs up to **100 iterations**. To change this:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Fix all linting errors" \
  --max-iterations 25
```

### Combined Setup and Spin

You can build the image and spin up a container in a single command using `--setup`:

```bash
spinner spin \
  --setup \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Add unit tests for the auth module"
```

This is equivalent to running `spinner setup --name my-sandbox` followed by `spinner spin --image my-sandbox ...`.

You can also pass `--base-image` or `--dockerfile` when using `--setup`:

```bash
spinner spin \
  --setup \
  --image node-env \
  --base-image node:20-bullseye \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Fix the build pipeline"
```

### Passing Secrets and Environment Variables

Use the `--env` flag to inject custom environment variables (API keys, tokens, config values) into the container at
runtime:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Publish the package" \
  --env NPM_TOKEN=npm_abc123 \
  --env MY_API_KEY=sk-xyz
```

The flag is repeatable — pass `--env` once per variable. Values are split on the first `=`, so values containing `=` are
handled correctly (e.g., `--env "DATABASE_URL=postgres://host/db?ssl=true"`).

Secrets passed via `--env` are written to a temporary file and delivered to Docker via `--env-file`. This keeps secret
values out of host process listings (`ps aux`). The temporary file is deleted immediately after the container starts.

> **Note:** You cannot override reserved variables (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `REPO_URL`, `PROMPT`,
> `BRANCH`, `MAX_ITERATIONS`, and others used internally). The CLI will print an error if you try.

## Step 3: Watch Progress

For autonomous runs, you will want to monitor what the agent is doing. There are two ways to enter watch mode.

### Using the `--watch` Flag

Add `--watch` when spinning up to immediately enter watch mode after the container starts:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Implement the new API endpoints" \
  --watch
```

### Using the `watch` Command

If the container is already running, use the `watch` command with the container name:

```bash
spinner watch <container-name>
```

For example:

```bash
spinner watch spinner-my-sandbox-your-repo-main
```

### What Watch Shows

The watch UI displays:

- **Container status** — whether it is running, stopped, or exited
- **Resource usage** — CPU and memory consumption
- **Streaming logs** — real-time output from the agent loop

Press `q` or `Ctrl+C` to exit watch mode. The container keeps running in the background.

## Managing Containers

### Reusing Containers

Spinner reuses containers by default. Running the same `spin` command (same image, repo, and branch) will reconnect to the existing container rather than creating a new one. This means the agent picks up where it left off.

### Recreating Containers

To start fresh, destroying the existing container and its state:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Start over on the auth refactor" \
  --recreate
```

### Checking State Manually

Spinner stores state and logs on your host machine at `~/.spinner/<container-name>/`:

```bash
# View iteration progress and status
cat ~/.spinner/<container-name>/state/state.json

# View raw logs
cat ~/.spinner/<container-name>/logs/raw.log
```

The state file tracks:

| Field           | Description                                                      |
|-----------------|------------------------------------------------------------------|
| `iteration`     | Current iteration number                                         |
| `status`        | `running`, `completed`, `rate_limited`, `error`, or `auth_error` |
| `started_at`    | When execution started                                           |
| `completed_at`  | When execution finished (if done)                                |
| `error_message` | Error details (if any)                                           |

### Stopping a Container

Containers are managed through standard Docker commands:

```bash
docker stop <container-name>
docker rm <container-name>
```

## Rate Limiting

If Claude hits an API rate limit during autonomous execution, Spinner automatically pauses for 61 minutes and then resumes. The iteration that was rate-limited is retried (it does not count against your iteration limit). You can see this reflected in the state file as `"status": "rate_limited"`.

## Quick Reference

| Task                   | Command                                                                |
|------------------------|------------------------------------------------------------------------|
| Build a sandbox image  | `spinner setup --name my-sandbox`                                      |
| Interactive sandbox    | `spinner spin --image my-sandbox --repo <url>`                         |
| Autonomous agent       | `spinner spin --image my-sandbox --repo <url> --prompt "task"`         |
| Watch live             | `spinner spin --image my-sandbox --repo <url> --prompt "task" --watch` |
| Watch existing         | `spinner watch <container-name>`                                       |
| Recreate container     | `spinner spin --image my-sandbox --repo <url> --recreate`              |
| Check state            | `cat ~/.spinner/<container-name>/state/state.json`                     |
| Pass secrets           | `spinner spin --image my-sandbox --repo <url> --env NPM_TOKEN=abc`     |
| Update spinner         | `spinner update`                                                       |
