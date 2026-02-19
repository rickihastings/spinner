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

# Helper script: runs as ExecStart — decrypts secrets and connects Tailscale.
# Using a separate script avoids systemd ExecStart quoting complexity.
cat > /usr/local/bin/tailscale-auth.sh << 'TSEOF'
#!/bin/bash
set -e
spinner secret inject -- sh -c '
    if [ -n "$TAILSCALE_AUTH_KEY" ]; then
        echo "Tailscale auth key found, connecting..."
        tailscale up --authkey="$TAILSCALE_AUTH_KEY" --ssh
    else
        echo "TAILSCALE_AUTH_KEY not set in spinner secrets, skipping"
    fi
'
TSEOF
chmod +x /usr/local/bin/tailscale-auth.sh

# Oneshot service that runs the helper once secrets are available.
cat > /etc/systemd/system/tailscale-auth.service << 'TSEOF'
[Unit]
Description=Tailscale auto-auth from spinner secret store
After=tailscaled.service
Wants=tailscaled.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/tailscale-auth.sh
RemainAfterExit=yes
TSEOF

# Watch for the ephemeral decryption key written by gcp_runtime.sh.
# When /run/spinner/secrets.key appears, systemd fires tailscale-auth.service automatically.
# This avoids any dependency on google-startup-scripts.service (which blocks until the
# agent loop finishes), while still guaranteeing secrets are available before auth runs.
cat > /etc/systemd/system/tailscale-auth.path << 'TSEOF'
[Unit]
Description=Watch for spinner secrets key

[Path]
PathExists=/run/spinner/secrets.key
Unit=tailscale-auth.service

[Install]
WantedBy=multi-user.target
TSEOF

systemctl enable tailscale-auth.path

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
