#!/bin/bash
set -e

# GCP Image Bake Script
# This script runs as a startup-script on a temporary GCP VM during `spinner setup --backend gcp`.
# It installs all tooling needed to run spinner agents, then shuts down so the VM's
# boot disk can be captured as a reusable custom image.

echo "=== Spinner GCP Image Bake ==="

# Install system dependencies
echo "Installing system dependencies..."
apt-get update
apt-get install -y git curl sudo ca-certificates jq unzip

# Install GitHub CLI
echo "Installing GitHub CLI..."
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null
apt-get update
apt-get install -y gh

# Create spinner user
echo "Creating spinner user..."
if ! id -u spinner &>/dev/null; then
    useradd -m -s /bin/bash spinner
    echo "spinner ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
else
    echo "User 'spinner' already exists, skipping creation."
fi
mkdir -p /home/spinner
chown -R spinner:spinner /home/spinner
chmod 755 /home/spinner

# Install Claude Code CLI
echo "Installing Claude Code CLI..."
su - spinner -c 'curl -fsSL https://claude.ai/install.sh | bash'

# Download and install spinner binary
# Get STATE_BUCKET and SPINNER_VERSION from instance metadata
export STATE_BUCKET=$(curl -sf -H "Metadata-Flavor: Google" \
    "http://metadata.google.internal/computeMetadata/v1/instance/attributes/STATE_BUCKET" || echo "")

# Read encrypted secrets blob and key (if provided via --secret on setup)
META_URL="http://metadata.google.internal/computeMetadata/v1/instance/attributes"
SPINNER_SECRET_BLOB_B64=$(curl -sf -H "Metadata-Flavor: Google" \
    "$META_URL/SPINNER_SECRET_BLOB" 2>/dev/null || echo "")
if [ -n "$SPINNER_SECRET_BLOB_B64" ] && [ -n "$STATE_BUCKET" ]; then
    mkdir -p /run/spinner
    echo "$SPINNER_SECRET_BLOB_B64" | base64 -d > /run/spinner/secrets.enc
    chmod 600 /run/spinner/secrets.enc
    KEY_PATH="gs://${STATE_BUCKET}/{{.ImageName}}-bake/secrets.key"
    if gsutil -q stat "$KEY_PATH" 2>/dev/null; then
        gsutil cp "$KEY_PATH" /run/spinner/secrets.key
        chmod 600 /run/spinner/secrets.key
        echo "Secrets available for bake script via: spinner secret inject"
    fi
fi

export SPINNER_VERSION=$(curl -sf -H "Metadata-Flavor: Google" \
    "http://metadata.google.internal/computeMetadata/v1/instance/attributes/SPINNER_VERSION" || echo "")

# Download and run the shared install script
curl -sf -H "Metadata-Flavor: Google" \
    "http://metadata.google.internal/computeMetadata/v1/instance/attributes/spinner-install-script" \
    > /tmp/install_spinner.sh
chmod +x /tmp/install_spinner.sh
/tmp/install_spinner.sh
rm /tmp/install_spinner.sh

# Set up workspace directory
echo "Setting up workspace..."
mkdir -p /home/spinner/workspace
chown spinner:spinner /home/spinner/workspace

# Set up log and state directories
mkdir -p /home/spinner/logs /home/spinner/state
chown spinner:spinner /home/spinner/logs /home/spinner/state

# Install startup.sh (passed via metadata during bake)
echo "Installing startup script..."
curl -sf -H "Metadata-Flavor: Google" \
    "http://metadata.google.internal/computeMetadata/v1/instance/attributes/startup-script-runtime" \
    > /usr/local/bin/startup.sh
chmod +x /usr/local/bin/startup.sh
{{if .BakeScript}}
# --- Custom bake script (injected via --bake-script flag) ---
echo "Running custom bake script..."
{{.BakeScript}}
echo "Custom bake script completed."
{{end}}
echo "=== Bake Complete ==="
echo "SPINNER_BAKE_COMPLETE" > /dev/ttyS0

# Shut down so the disk can be captured as an image
shutdown -h now
