#!/bin/bash
set -e

# Shared script to install the spinner binary
# Used by both Docker (extending.template) and GCP (gcp_bake.sh)
#
# Environment variables:
#   LOCAL_BUILD - if "true", download from state bucket instead of GitHub
#   STATE_BUCKET - GCS bucket containing local dev binary (required if LOCAL_BUILD=true)
#   SPINNER_VERSION - if set, download this specific version instead of latest (e.g. "v1.0.0")

echo "Installing spinner binary..."

# Check for local development mode
if [ "${LOCAL_BUILD}" = "true" ] && [ -n "${STATE_BUCKET}" ]; then
    echo "LOCAL_BUILD detected, downloading from state bucket..."

    # Download from state bucket
    if gsutil cp "gs://${STATE_BUCKET}/local-dev/spinner-dev-linux-amd64.tar.gz" /tmp/spinner.tar.gz 2>/dev/null; then
        tar -xzf /tmp/spinner.tar.gz -C /usr/local/bin
        chmod +x /usr/local/bin/spinner
        rm /tmp/spinner.tar.gz
        echo "✅ Installed local development spinner"
    else
        echo "Error: LOCAL_BUILD=true but no binary found in gs://${STATE_BUCKET}/local-dev/"
        echo "Make sure the setup command uploaded the binary first"
        exit 1
    fi
elif [ "${LOCAL_BUILD}" = "true" ] && [ -f "/tmp/spinner" ] && [ -s "/tmp/spinner" ]; then
    # Docker mode: binary was COPYed to /tmp/spinner
    # -s checks that file exists and is not empty (not a placeholder)
    echo "LOCAL_BUILD detected, using local binary..."
    mv /tmp/spinner /usr/local/bin/spinner
    chmod +x /usr/local/bin/spinner
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
        VERSION_NUM="${SPINNER_VERSION#v}"
        curl -fsSL "https://github.com/rickihastings/spinner/releases/download/${SPINNER_VERSION}/spinner_${VERSION_NUM}_linux_amd64.tar.gz" -o /tmp/spinner.tar.gz
        tar -xzf /tmp/spinner.tar.gz -C /usr/local/bin
        chmod +x /usr/local/bin/spinner
        rm /tmp/spinner.tar.gz
        echo "✅ Installed spinner ${SPINNER_VERSION} from GitHub"
    fi
fi
