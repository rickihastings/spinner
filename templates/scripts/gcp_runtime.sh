#!/bin/bash
set -e

# GCP Runtime Startup Script
# This script runs on each VM boot during `spinner spin --backend gcp`.
# It reads configuration from instance metadata, sets up the environment,
# and delegates to the standard startup.sh for repo cloning and execution.

echo "=== Spinner GCP Runtime ==="

# Read configuration from instance metadata
META_URL="http://metadata.google.internal/computeMetadata/v1/instance/attributes"
META_HEADER="Metadata-Flavor: Google"

echo "Reading instance metadata..."

GITHUB_TOKEN=$(curl -sf -H "$META_HEADER" "$META_URL/GITHUB_TOKEN" || echo "")
CLAUDE_CODE_OAUTH_TOKEN=$(curl -sf -H "$META_HEADER" "$META_URL/CLAUDE_CODE_OAUTH_TOKEN" || echo "")
REPO_URL=$(curl -sf -H "$META_HEADER" "$META_URL/REPO_URL" || echo "")
PROMPT=$(curl -sf -H "$META_HEADER" "$META_URL/PROMPT" || echo "")
BRANCH=$(curl -sf -H "$META_HEADER" "$META_URL/BRANCH" || echo "")
MAX_ITERATIONS=$(curl -sf -H "$META_HEADER" "$META_URL/MAX_ITERATIONS" || echo "100")
SPINNER_LOG_BUCKET=$(curl -sf -H "$META_HEADER" "$META_URL/SPINNER_LOG_BUCKET" || echo "")
SPINNER_INSTANCE_NAME=$(curl -sf -H "$META_HEADER" "$META_URL/SPINNER_INSTANCE_NAME" || echo "")

SPINNER_STATE_BUCKET=$(curl -sf -H "$META_HEADER" "$META_URL/SPINNER_STATE_BUCKET" || echo "")

export GITHUB_TOKEN CLAUDE_CODE_OAUTH_TOKEN REPO_URL PROMPT BRANCH MAX_ITERATIONS
export SPINNER_LOG_BUCKET SPINNER_INSTANCE_NAME SPINNER_STATE_BUCKET

# Read custom env vars from metadata (SPINNER_ENV_ prefix)
echo "Reading custom environment variables..."
for key in $(curl -sf -H "$META_HEADER" "$META_URL/" 2>/dev/null | grep "^SPINNER_ENV_" || echo ""); do
    if [ -n "$key" ]; then
        value=$(curl -sf -H "$META_HEADER" "$META_URL/$key" 2>/dev/null || echo "")
        # Strip SPINNER_ENV_ prefix and export
        var_name="${key#SPINNER_ENV_}"
        export "$var_name=$value"
        echo "Exported custom var: $var_name"
    fi
done

# Set log directory and state directory
export LOG_DIR="/home/spinner/logs"
export STATE_DIR="/home/spinner/state"

# Download state from GCS if available
if [ -n "$SPINNER_STATE_BUCKET" ] && [ -n "$SPINNER_INSTANCE_NAME" ]; then
    echo "Checking for existing state in GCS..."
    STATE_PATH="gs://${SPINNER_STATE_BUCKET}/${SPINNER_INSTANCE_NAME}/state.json"
    mkdir -p /home/spinner/state
    if gsutil -q stat "$STATE_PATH" 2>/dev/null; then
        gsutil cp "$STATE_PATH" /home/spinner/state/state.json
        chown spinner:spinner /home/spinner/state/state.json
        echo "State restored from GCS."
    else
        echo "No existing state found, starting fresh."
    fi
fi

# Copy the startup.sh script from the baked image location
# The bake script installs startup.sh into the image at /usr/local/bin/startup.sh
# If it doesn't exist there, fall back to using the bundled template
STARTUP_SCRIPT="/usr/local/bin/startup.sh"
if [ ! -f "$STARTUP_SCRIPT" ]; then
    echo "Warning: startup.sh not found at $STARTUP_SCRIPT"
    exit 1
fi

echo "Delegating to startup.sh..."

# Disable set -e so we can capture the exit code from startup.sh
set +e

# Switch to spinner user and run startup in the workspace directory
# Use su -m to preserve environment variables (including custom ones)
su -m spinner -c "export HOME=/home/spinner && export PATH=\"/home/spinner/.claude/bin:\$PATH\" && cd /home/spinner/workspace && \
    /usr/local/bin/startup.sh"
EXIT_CODE=$?

# Re-enable set -e
set -e

# If spinner exec completed successfully (exit 0) AND a prompt was specified, shutdown VM
# Without a prompt, the VM should keep running for interactive use
if [ $EXIT_CODE -eq 0 ] && [ -n "$PROMPT" ]; then
    echo "Execution completed successfully. Stopping VM instance..."
    # Give logs/state a moment to flush
    sleep 2
    # Initiate graceful shutdown
    sudo poweroff
fi

 # For non-zero exit codes, write error state to GCS so watchers can detect the failure
if [ $EXIT_CODE -ne 0 ] && [ -n "$SPINNER_STATE_BUCKET" ] && [ -n "$SPINNER_INSTANCE_NAME" ]; then
    echo "Startup failed with exit code $EXIT_CODE. Writing error state to GCS..."
    STATE_JSON=$(cat <<STATEEOF
{
  "iteration": 0,
  "status": "error",
  "error_message": "startup script failed with exit code $EXIT_CODE",
  "last_updated": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "started_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
STATEEOF
)
    echo "$STATE_JSON" | gsutil -q cp - "gs://${SPINNER_STATE_BUCKET}/${SPINNER_INSTANCE_NAME}/state.json" || true
fi

# For non-zero exit codes or when no prompt specified, keep VM running
exit $EXIT_CODE
