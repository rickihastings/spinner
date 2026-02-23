#!/bin/bash
set -e

# Shared script to install the spinner binary
# Used by both Docker (extending.template) and GCP (gcp_bake.sh)
#
# Detection order:
#   1. /tmp/spinner exists and is non-empty → use it (Docker dev build)
#   2. STATE_BUCKET is set and dev tarball exists in GCS → use it (GCP dev build)
#   3. Download from GitHub releases (production)
#
# Environment variables:
#   STATE_BUCKET - GCS bucket containing local dev binary (optional, for GCP dev)
#   SPINNER_VERSION - if set, download this specific version instead of latest (e.g. "v1.0.0")

echo "Installing spinner binary..."

# Check for Docker dev mode: binary was COPYed to /tmp/spinner
if [ -f "/tmp/spinner" ] && [ -s "/tmp/spinner" ]; then
    # -s checks that file exists and is not empty (not a placeholder)
    echo "Local development binary detected at /tmp/spinner..."
    mv /tmp/spinner /usr/local/bin/spinner
    chmod +x /usr/local/bin/spinner
    echo "✅ Installed local development spinner"
# Check for GCP dev mode: download from state bucket
elif [ -n "${STATE_BUCKET}" ] && gsutil cp "gs://${STATE_BUCKET}/local-dev/spinner-dev-linux-amd64.tar.gz" /tmp/spinner.tar.gz 2>/dev/null; then
    echo "Local development binary detected in state bucket..."
    tar -xzf /tmp/spinner.tar.gz -C /usr/local/bin
    chmod +x /usr/local/bin/spinner
    rm /tmp/spinner.tar.gz
    echo "✅ Installed local development spinner"
else
    # Production mode: download from GitHub releases
    # SPINNER_VERSION can be passed from the host to ensure version parity
    if [ -z "$SPINNER_VERSION" ]; then
        echo "Downloading latest spinner from GitHub releases..."
        SPINNER_VERSION=$(curl -sf https://api.github.com/repos/rickihastings/spinner/releases/latest | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)
    else
        echo "Downloading spinner ${SPINNER_VERSION} (pinned to host version)..."
    fi

    if [ -z "$SPINNER_VERSION" ]; then
        echo "Warning: Could not detect latest release, skipping spinner install"
    else
        curl -fsSL "https://github.com/rickihastings/spinner/releases/download/${SPINNER_VERSION}/spinner_linux_amd64.tar.gz" -o /tmp/spinner.tar.gz
        tar -xzf /tmp/spinner.tar.gz -C /usr/local/bin
        chmod +x /usr/local/bin/spinner
        rm /tmp/spinner.tar.gz
        echo "✅ Installed spinner ${SPINNER_VERSION} from GitHub"
    fi
fi
