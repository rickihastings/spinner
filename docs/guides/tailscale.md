# Tailscale Guide

Tailscale lets you SSH into your GCP VMs securely from anywhere — including mobile — without exposing public IPs or
configuring firewall rules. Once a VM joins your Tailscale network, you can reach it from any device on that same
network using its Tailscale IP (`100.x.x.x`).

This is particularly useful for:

- SSHing into interactive VMs from your phone (e.g. via [Terminus](https://www.termius.com/))
- Checking in on autonomous agent runs without needing `gcloud` CLI
- Remote access from networks that block outbound SSH (port 22)

## How It Works

During the bake phase, Tailscale is installed and a systemd service is configured to authenticate on first boot. When
you spin up a VM, you pass your Tailscale auth key via `--secret`. The service decrypts and reads the key using
`spinner secret inject` and connects the VM to your Tailscale network automatically.

Tailscale's `--ssh` flag is used, which enables [Tailscale SSH](https://tailscale.com/kb/1193/tailscale-ssh). This
means Tailscale handles SSH authentication — you do not need to manage SSH key pairs separately. You can configure
which users can connect and as which UNIX user via [Tailscale ACLs](https://login.tailscale.com/admin/acls).

## Step 1: Create a Tailscale Auth Key

1. Go to [https://login.tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys)
2. Click **Generate auth key**
3. Enable **Reusable** (so multiple VMs can use the same key) and **Ephemeral** (so VMs are removed from your network
   when they stop)
4. Copy the key — it starts with `tskey-auth-`

## Step 2: Bake Tailscale into Your Image

Create a bake script that installs Tailscale and registers a systemd service to connect on boot:

```bash
#!/bin/bash
# tailscale-bake.sh

set -e

echo "=== Installing Tailscale ==="
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/jammy.noarmor.gpg \
    -o /usr/share/keyrings/tailscale-archive-keyring.gpg
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/jammy.tailscale-keyring.list \
    -o /etc/apt/sources.list.d/tailscale.list

apt-get update
apt-get install -y tailscale

systemctl enable tailscaled

# Create a oneshot service that reads TAILSCALE_AUTHKEY from the Spinner secret store
# and authenticates on boot. If the key is absent, it does nothing.
cat > /etc/systemd/system/tailscale-auth.service << 'TSEOF'
[Unit]
Description=Tailscale auto-auth via Spinner secrets
After=tailscaled.service google-startup-scripts.service
Wants=tailscaled.service

[Service]
Type=oneshot
User=spinner
ExecStart=/bin/bash -c '\
    if [ -f /run/spinner/secrets.enc ] && [ -f /run/spinner/secrets.key ]; then \
        echo "Tailscale auth key found, connecting..."; \
        spinner secret inject -- sh -c "tailscale up --authkey=\"\$TAILSCALE_AUTHKEY\" --ssh"; \
    else \
        echo "No Spinner secrets available, skipping Tailscale auto-auth"; \
    fi'
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
TSEOF

systemctl enable tailscale-auth.service

echo "Tailscale installed: $(tailscale version)"
```

Then bake the image:

```bash
spinner setup --name my-sandbox --bake-script ./tailscale-bake.sh
```

## Step 3: Spin with Your Auth Key

Store your Tailscale auth key in Spinner's encrypted secret store, then reference it with `--secret`:

```bash
# Store the key once
spinner secret set TAILSCALE_AUTHKEY tskey-auth-xxxxx

# Interactive VM
spinner spin \
  --image my-sandbox \
  --repo https://github.com/your-org/your-repo.git \
  --secret TAILSCALE_AUTHKEY

# Autonomous agent
spinner spin \
  --image my-sandbox \
  --repo https://github.com/your-org/your-repo.git \
  --prompt "Fix the authentication bug" \
  --secret TAILSCALE_AUTHKEY
```

Within a few seconds of the VM starting, it will appear in your [Tailscale admin console](https://login.tailscale.com/admin/machines)
under its GCP instance name.

## Step 4: Find the VM's Tailscale IP

Once the VM is running, find its Tailscale IP in one of two ways:

**From the Tailscale admin console:**

Go to [https://login.tailscale.com/admin/machines](https://login.tailscale.com/admin/machines) and look for the VM by
name. Its Tailscale IP (`100.x.x.x`) is shown next to the machine entry.

**From the VM directly:**

```bash
gcloud compute ssh <instance-name> --zone <zone> --project <project> -- tailscale ip -4
```

## Step 5: Connect via Terminus

[Terminus](https://www.termius.com/) is an SSH client for iOS and Android. To connect to your VM:

1. Install Terminus from the App Store or Google Play
2. Tap **+** → **New Host**
3. Fill in the connection details:
   - **Hostname**: the Tailscale IP (`100.x.x.x`)
   - **Port**: `22`
   - **Username**: your Tailscale username (the part before `@` in your Tailscale login email)
4. Make sure Tailscale is installed and connected on your phone
5. Tap the host to connect

> **Tailscale ACLs**: Tailscale SSH maps the connecting user's Tailscale identity to a UNIX user. By default it allows
> login as your Tailscale username. If the `spinner` UNIX user does not match your Tailscale identity, configure SSH
> access rules in your [Tailscale ACL policy](https://login.tailscale.com/admin/acls). For example, to allow your
> identity to log in as `root`:
>
> ```json
> "ssh": [
>   {
>     "action": "accept",
>     "src": ["autogroup:members"],
>     "dst": ["autogroup:self"],
>     "users": ["autogroup:nonroot", "root"]
>   }
> ]
> ```

## Alternative: Traditional SSH Keys

If you prefer standard SSH key authentication instead of Tailscale SSH, add your public key to the `spinner` user's
`authorized_keys` in your bake script:

```bash
#!/bin/bash
# Add your SSH public key during bake (Tailscale still provides the network path)

mkdir -p /home/spinner/.ssh
echo "ssh-ed25519 AAAA... your-key-comment" >> /home/spinner/.ssh/authorized_keys
chmod 700 /home/spinner/.ssh
chmod 600 /home/spinner/.ssh/authorized_keys
chown -R spinner:spinner /home/spinner/.ssh
```

Then in Terminus, set the username to `spinner` and attach your private key in the Keychain settings.

## Quick Reference

| Task                        | Command / Action                                                    |
|-----------------------------|---------------------------------------------------------------------|
| Generate auth key           | [tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys) |
| Bake image with Tailscale   | `spinner setup --name my-sandbox --bake-script ./tailscale-bake.sh` |
| Spin with Tailscale         | `spinner spin --image my-sandbox --repo <url> --secret TAILSCALE_AUTHKEY`             |
| Find Tailscale IP           | [tailscale.com/admin/machines](https://login.tailscale.com/admin/machines) |
| Check Tailscale status      | `gcloud compute ssh <instance> -- tailscale status`                 |
| Connect from Terminus       | Host: Tailscale IP, Port: 22, User: your Tailscale username         |
