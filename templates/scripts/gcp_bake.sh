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
useradd -m -s /bin/bash spinner
echo "spinner ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Install Claude Code CLI
echo "Installing Claude Code CLI..."
su - spinner -c 'curl -fsSL https://claude.ai/install.sh | bash'

# Download spinner binary from latest GitHub Release
echo "Installing spinner binary..."
SPINNER_VERSION=$(curl -sf https://api.github.com/repos/rickihastings/spinner/releases/latest \
    | jq -r '.tag_name')

if [ -z "$SPINNER_VERSION" ] || [ "$SPINNER_VERSION" = "null" ]; then
    echo "Warning: Could not detect latest spinner release, skipping binary install"
else
    VERSION_NUM="${SPINNER_VERSION#v}"
    curl -fsSL "https://github.com/rickihastings/spinner/releases/download/${SPINNER_VERSION}/spinner_${VERSION_NUM}_linux_amd64.tar.gz" \
        -o /tmp/spinner.tar.gz
    tar -xzf /tmp/spinner.tar.gz -C /usr/local/bin spinner
    chmod +x /usr/local/bin/spinner
    rm /tmp/spinner.tar.gz
    echo "Installed spinner ${SPINNER_VERSION}"
fi

# Set up workspace directory
echo "Setting up workspace..."
mkdir -p /home/spinner/workspace
chown spinner:spinner /home/spinner/workspace

# Set up log and state directories
mkdir -p /home/spinner/logs /home/spinner/state
chown spinner:spinner /home/spinner/logs /home/spinner/state

echo "=== Bake Complete ==="
echo "SPINNER_BAKE_COMPLETE" > /dev/ttyS0

# Shut down so the disk can be captured as an image
shutdown -h now
