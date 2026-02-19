#!/bin/bash
# Custom bake script for spinner development on GCP.
# This script is injected into the base bake image via --bake-script.
# It runs as root; the 'spinner' user already exists.

set -e

echo "=== Installing Go 1.24.7 ==="
curl -fsSL https://go.dev/dl/go1.24.7.linux-amd64.tar.gz -o /tmp/go.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz

# System-wide PATH for Go
cat > /etc/profile.d/golang.sh << 'GOEOF'
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
export GOPATH="$HOME/go"
GOEOF
chmod +x /etc/profile.d/golang.sh

# Set up GOPATH and PATH for spinner user
su - spinner -c 'mkdir -p ~/go/bin'

# Add Go to spinner user's shell profiles so it's available in all shell types
# (profile.d only works for login shells, not su -m or non-interactive bash)
GO_PATH_EXPORT='export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
export GOPATH="$HOME/go"'
echo "$GO_PATH_EXPORT" >> /home/spinner/.bashrc
echo "$GO_PATH_EXPORT" >> /home/spinner/.profile

# Verify
/usr/local/go/bin/go version

echo "=== Installing Docker CE ==="
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu jammy stable" \
    > /etc/apt/sources.list.d/docker.list

apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin

usermod -aG docker spinner
systemctl enable docker

echo "=== Installing Tailscale ==="
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/jammy.noarmor.gpg \
    -o /usr/share/keyrings/tailscale-archive-keyring.gpg
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/jammy.tailscale-keyring.list \
    -o /etc/apt/sources.list.d/tailscale.list

apt-get update
apt-get install -y tailscale

systemctl enable tailscaled

# Create a oneshot service that reads TAILSCALE_AUTHKEY from GCE instance metadata
# and authenticates on boot. If the key is absent, it does nothing.
cat > /etc/systemd/system/tailscale-auth.service << 'TSEOF'
[Unit]
Description=Tailscale auto-auth from GCE metadata
After=tailscaled.service
Wants=tailscaled.service

[Service]
Type=oneshot
ExecStart=/bin/bash -c '\
    KEY=$(curl -sf -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/attributes/TAILSCALE_AUTHKEY" || true); \
    if [ -n "$KEY" ]; then \
        echo "Tailscale auth key found, connecting..."; \
        tailscale up --authkey="$KEY" --ssh; \
    else \
        echo "No TAILSCALE_AUTHKEY in metadata, skipping auto-auth"; \
    fi'
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
TSEOF

systemctl enable tailscale-auth.service

echo "=== Installing Node.js 22.x LTS ==="
curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
apt-get install -y nodejs

echo "=== Installing Claude Code CLI and OpenSpec (global) ==="
npm install -g @anthropic-ai/claude-code @fission-ai/openspec

echo "=== Installing golangci-lint ==="
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b /usr/local/bin

# Verify installations
echo "=== Verifying installations ==="
/usr/local/go/bin/go version
docker --version
tailscale version
node --version
npm --version
golangci-lint --version

echo "=== Dev bake script complete ==="
