# GCP Sandbox Guide

This guide walks you through using Spinner to run Claude agents on Google Cloud Platform VMs. You can either create a VM
to work in interactively, or launch an autonomous agent loop that works on a task unsupervised.

## Prerequisites

Before you begin, make sure you have:

- **Google Cloud SDK** installed (`gcloud` CLI available)
- **A GCP project** with billing enabled
- **A GitHub token** exported as `GITHUB_TOKEN`
- **A Claude Code OAuth token** exported as `CLAUDE_CODE_OAUTH_TOKEN`
- **GCP authentication** configured (see below)

```bash
export GITHUB_TOKEN="ghp_your_token_here"
export CLAUDE_CODE_OAUTH_TOKEN="your_oauth_token_here"
```

### GCP Authentication

Authenticate using either method:

```bash
# Option A: Application Default Credentials (recommended for development)
gcloud auth application-default login

# Option B: Service account key file
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

### Bootstrap GCP Resources

Run the bootstrap script to automatically provision the required GCP resources (GCS bucket, service account, IAM roles):

```bash
scripts/gcp-bootstrap.sh --project my-project --zone us-central1-a
```

The script is idempotent and safe to re-run. It creates:

- A GCS bucket for state persistence (`{project}-spinner-state` by default)
- A service account with the required IAM roles
- Enables the Compute Engine, Storage, Logging, and Monitoring APIs

Run `scripts/gcp-bootstrap.sh --help` for all options.

After running the script, export the variables it prints:

```bash
export SPINNER_BACKEND=gcp
export SPINNER_PROJECT=my-project
export SPINNER_ZONE=us-central1-a
export SPINNER_STATE_BUCKET=my-project-spinner-state
```

> **Tip:** Add these to your shell profile or use a `.spinner.json` config file (see [Configuration](#configuration)
> below) so you do not need to export them every time.

## Step 1: Bake a VM Image

Before spinning up VMs, you need to bake a GCP image. This image comes pre-configured with git, GitHub CLI, Claude Code,
and the Spinner binary.

```bash
spinner setup --name my-sandbox
```

This creates a temporary VM from `ubuntu-2204-lts`, installs all tooling, shuts down the VM, and captures its disk as a
reusable custom image. The process takes around 5-10 minutes.

### Custom Bake Scripts

To install additional tools or dependencies into the image, use the `--bake-script` flag. This is the GCP equivalent of
a custom Dockerfile.

Create a shell script with your custom installation steps:

```bash
# custom-install.sh
#!/bin/bash

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt-get install -y nodejs

# Install Python and pip
apt-get install -y python3 python3-pip

# Install project-specific tools
npm install -g typescript eslint
pip3 install black pytest
```

Then pass it during setup:

```bash
spinner setup --name node-env --bake-script ./custom-install.sh
```

The custom script runs as root during the bake process, after the core tooling (git, Claude Code, Spinner) is installed
but before the VM shuts down. You can install any packages, configure services, or set up the environment however you
need.

> **Note:** Since baking creates a reusable image, anything you install via `--bake-script` is available on every VM
> spun from that image. This avoids reinstalling dependencies on each run.

### Customizing Claude Code Configuration

If you use custom MCP servers, slash commands, skills, or other Claude Code settings, you can include them in the baked
image via your bake script. The bake script runs as root after the `spinner` user and Claude Code are already installed,
so you can write directly to `/home/spinner/.claude/`.

**Inline configuration:**

```bash
#!/bin/bash

# Create the .claude config directory
mkdir -p /home/spinner/.claude/commands

# Write MCP server and settings configuration
cat > /home/spinner/.claude/settings.json << 'EOF'
{
  "permissions": {
    "allow": ["mcp__my-server__*"]
  },
  "mcpServers": {
    "my-server": {
      "command": "my-mcp-server",
      "args": ["--port", "3000"]
    }
  }
}
EOF

# Write a custom slash command
cat > /home/spinner/.claude/commands/deploy.md << 'EOF'
Run the deployment pipeline and verify it succeeds.
EOF

# Fix ownership
chown -R spinner:spinner /home/spinner/.claude/
```

**Download from a GCS bucket:**

If you maintain your Claude Code configuration in a GCS bucket:

```bash
#!/bin/bash

# Download Claude Code config from GCS
gsutil -m cp -r gs://my-config-bucket/claude-config/ /home/spinner/.claude/
chown -R spinner:spinner /home/spinner/.claude/
```

**Clone from a git repository:**

```bash
#!/bin/bash

# Clone config repo and copy Claude Code settings
git clone https://github.com/my-org/agent-config.git /tmp/agent-config
cp -r /tmp/agent-config/.claude/ /home/spinner/.claude/
chown -R spinner:spinner /home/spinner/.claude/
rm -rf /tmp/agent-config
```

Then bake the image:

```bash
spinner setup --name my-custom-env --bake-script ./custom-install.sh
```

> **Tip:** Only include the specific config files you need (settings, commands, skills) rather than your entire
> `~/.claude/` directory, which may contain tokens or session data that should not be baked into an image.

### Customizing VM Resources

You can configure the machine type and disk size:

```bash
spinner setup --name powerful-env \
  --machine-type e2-standard-8 \
  --disk-size 50
```

| Flag             | Default         | Description                                              |
|------------------|-----------------|----------------------------------------------------------|
| `--machine-type` | `e2-standard-2` | GCP machine type for the bake VM and runtime VMs         |
| `--disk-size`    | `30`            | Boot disk size in GB                                     |
| `--bake-script`  | —               | Path to a custom shell script to run during image baking |

## Step 2: Spin Up a VM

Once your image is baked, you spin up a VM pointed at a git repository. How the VM behaves depends on whether you
provide a `--prompt`.

### Interactive Mode (No Prompt)

To create a VM you can SSH into and use manually:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --branch main
```

This starts a VM with your repository cloned and checked out. The VM stays running, and you can SSH into it:

```bash
gcloud compute ssh <instance-name> --zone <zone> --project <project>
```

From inside the VM, you have a full development environment with git configured and Claude Code available. Run Claude
interactively, install dependencies, or do whatever you need.

> **Tip:** The instance name is deterministic based on your image and repo. For example, with `--image my-sandbox` and a
> repo named `your-repo`, the instance name will be something like `spinner-my-sandbox-your-repo`. If you also pass
`--branch main`, it becomes `spinner-my-sandbox-your-repo-main`.

### Autonomous Mode (With Prompt)

To kick off an autonomous agent loop that works on a task without supervision:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --branch main \
  --prompt "Refactor the authentication module to use JWT tokens"
```

When a prompt is provided, Spinner launches an iteration loop inside the VM. Each iteration:

1. Runs Claude with your prompt
2. Checks the output for completion, errors, or rate limits
3. Pushes any changes to git
4. Saves progress to a state file (persisted to GCS)
5. Repeats until the task is done or the iteration limit is reached

The agent signals completion by outputting `~~ FEATURE_COMPLETED ~~`. At that point, the loop stops, your changes are on
the branch, and the VM automatically shuts down.

By default, the loop runs up to **100 iterations**. To change this:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Fix all linting errors" \
  --max-iterations 25
```

### Combined Setup and Spin

You can bake the image and spin up a VM in a single command using `--setup`:

```bash
spinner spin \
  --setup \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Add unit tests for the auth module"
```

You can also pass `--bake-script` when using `--setup`:

```bash
spinner spin \
  --setup \
  --image node-env \
  --bake-script ./custom-install.sh \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Fix the build pipeline"
```

### Passing Secrets and Environment Variables

Use the `--env` flag to inject custom environment variables (API keys, tokens, config values) into the VM at runtime:

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

On GCP, custom env vars are passed as instance metadata with a `SPINNER_ENV_` prefix (e.g., `SPINNER_ENV_NPM_TOKEN`).
The VM's runtime script automatically strips the prefix and exports the variables into the execution environment.

> **Note:** You cannot override reserved variables (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `REPO_URL`, `PROMPT`,
> `BRANCH`, `MAX_ITERATIONS`, and others used internally). The CLI will print an error if you try.

## Step 3: Watch Progress

For autonomous runs, you will want to monitor what the agent is doing.

### Using the `--watch` Flag

Add `--watch` when spinning up to immediately enter watch mode after the VM starts:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Implement the new API endpoints" \
  --watch
```

### Using the `watch` Command

If the VM is already running, use the `watch` command with the instance name:

```bash
spinner watch <instance-name>
```

### What Watch Shows

The watch UI displays:

- **Instance status** — whether it is running, stopped, or terminated
- **Resource usage** — CPU and memory consumption (via Google Cloud Ops Agent)
- **Streaming logs** — real-time output from the agent loop

Press `q` or `Ctrl+C` to exit watch mode. The VM keeps running in the background.

## Auto-Stop Behavior

GCP VMs automatically stop when the agent completes successfully **and a prompt was specified**. This prevents wasted
compute resources and reduces costs for autonomous tasks.

| Scenario                             | VM Behavior                              |
|--------------------------------------|------------------------------------------|
| Successful completion with prompt    | VM stops automatically within ~5 seconds |
| Successful completion without prompt | VM keeps running (interactive use)       |
| Errors or rate limits                | VM keeps running (for debugging)         |

To restart a stopped VM:

```bash
gcloud compute instances start <instance-name> \
  --zone <zone> \
  --project <project>
```

## Managing VMs

### Reusing VMs

Spinner reuses VMs by default. Running the same `spin` command (same image, repo, and branch) will reconnect to the
existing VM rather than creating a new one. State is restored from GCS, so the agent picks up where it left off.

### Recreating VMs

To start fresh, destroying the existing VM and its state:

```bash
spinner spin \
  --image my-sandbox \
  --repo git@github.com:your-org/your-repo.git \
  --prompt "Start over on the auth refactor" \
  --recreate
```

### Checking State

Spinner persists state to Google Cloud Storage for durability across VM lifecycle events:

```bash
# View iteration progress and status
gsutil cat gs://<state-bucket>/<instance-name>/state.json
```

The state file tracks:

| Field           | Description                                                      |
|-----------------|------------------------------------------------------------------|
| `iteration`     | Current iteration number                                         |
| `status`        | `running`, `completed`, `rate_limited`, `error`, or `auth_error` |
| `started_at`    | When execution started                                           |
| `completed_at`  | When execution finished (if done)                                |
| `error_message` | Error details (if any)                                           |

## Rate Limiting

If Claude hits an API rate limit during autonomous execution, Spinner automatically pauses for 61 minutes and then
resumes. The iteration that was rate-limited is retried (it does not count against your iteration limit). You can see
this reflected in the state file as `"status": "rate_limited"`.

## Configuration

Instead of exporting environment variables every time, you can use a `.spinner.json` config file in the directory where
you run Spinner:

```json
{
  "backend": "gcp",
  "project": "my-project",
  "zone": "us-central1-a",
  "state-bucket": "my-project-spinner-state",
  "machine-type": "e2-standard-4",
  "disk-size": 50
}
```

For local overrides that should not be committed, use a `.env` file:

```env
SPINNER_PROJECT=my-local-project
SPINNER_ZONE=us-west1-a
```

Configuration precedence (highest to lowest):

1. Command-line flags
2. Environment variables (`SPINNER_*`)
3. `.env` file
4. `.spinner.json` file
5. Default values

## Monitoring

The GCP backend automatically installs
the [Google Cloud Ops Agent](https://cloud.google.com/stackdriver/docs/solutions/agents/ops-agent) during image baking.
This provides:

- **CPU metrics** — real-time CPU utilization (visible in watch mode)
- **Memory metrics** — real-time memory usage percentage (visible in watch mode)
- **System metrics** — additional metrics available in the Cloud Monitoring console

No additional configuration is required. If the Ops Agent is not running, watch mode will display "N/A" for memory
metrics while CPU metrics continue to work via the standard Compute Engine metrics API.

## Quick Reference

| Task                    | Command                                                                            |
|-------------------------|------------------------------------------------------------------------------------|
| Bootstrap GCP resources | `scripts/gcp-bootstrap.sh --project <id> --zone <zone>`                            |
| Bake a VM image         | `spinner setup --name my-sandbox`                                                  |
| Bake with custom script | `spinner setup --name my-sandbox --bake-script ./install.sh`                       |
| Interactive VM          | `spinner spin --image my-sandbox --repo <url>`                                     |
| Autonomous agent        | `spinner spin --image my-sandbox --repo <url> --prompt "task"`                     |
| Watch live              | `spinner spin --image my-sandbox --repo <url> --prompt "task" --watch`             |
| Watch existing          | `spinner watch <instance-name>`                                                    |
| Pass secrets            | `spinner spin --image my-sandbox --repo <url> --env NPM_TOKEN=abc`                 |
| Recreate VM             | `spinner spin --image my-sandbox --repo <url> --recreate`                          |
| Check state             | `gsutil cat gs://<bucket>/<instance-name>/state.json`                              |
| SSH into VM             | `gcloud compute ssh <instance-name> --zone <zone> --project <project>`             |
| Restart stopped VM      | `gcloud compute instances start <instance-name> --zone <zone> --project <project>` |
| Update spinner          | `spinner update`                                                                   |