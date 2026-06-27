# Aurion Proxy Server

This repository contains the Aurion Proxy, a high-performance SMTP proxy written in Go.

It intercepts incoming SMTP traffic, queries the Aurion Core Server for routing instructions and key validation, processes/encrypts messages if necessary, and forwards them to the underlying mail backend.

## 1. Requirements

* Go 1.22+
* Git
* Port `25` available (requires root/sudo privileges to bind on most systems)

## 2. Environment Variables (`.env`)

Clone the repository and create your configuration file:

```bash
git clone https://github.com/aurion/proxy.git
cd proxy
cp .env.example .env

```

Edit the `.env` file with your specific configuration:

```ini
# SMTP proxy listener
LISTEN_ADDR=:25
DOMAIN=aurion.example.com
MAX_MESSAGE_BYTES=10485760   # 10 MiB

# Routing API (points to your Aurion Core Server instance)
ROUTING_URL=http://localhost:8080
ROUTING_TIMEOUT=3s

# Forwarding SMTP (where to send the processed mail, e.g., Stalwart/Postfix)
FORWARD_ADDR=127.0.0.1:10025

# Queue & Workers
QUEUE_SIZE=1000
WORKER_COUNT=4

# TLS configuration for incoming SMTP traffic
TLS_CERT=/etc/ssl/certs/aurion-proxy.crt
TLS_KEY=/etc/ssl/private/aurion-proxy.key

```



## 3. Development Mode

Install dependencies and run the proxy locally.

> Note: If you use port `:25`, you will need to run this command with `sudo`. For local development without root privileges, you can change `LISTEN_ADDR` to `:2525` in your `.env`.

```bash
go mod tidy
sudo --preserve-env=LISTEN_ADDR go run ./cmd/aurion-proxy

```

## 4. Production Installation (Standard Usage)

For production deployments, compile the application into a standalone binary and run it as a system service.

### Step 1: Build the Binary

Compile the proxy with optimized flags:

```bash
go mod tidy
go build -ldflags="-w -s" -o aurion-proxy ./cmd/aurion-proxy

```

### Step 2: Deployment

Move the compiled binary and your `.env` configuration file to their production directory (e.g., `/var/www/aurion-proxy`).

Ensure you generate or copy your SSL/TLS certificates (`TLS_CERT` and `TLS_KEY`) to the paths defined in your `.env`.

### Step 2 Bis: Certificates

If your Aurion Core Server (Web API) and Aurion Proxy (SMTP) are hosted on the same machine, they can share the exact same domain name (e.g., `aurion.example.com`) and the same SSL certificate because they listen on different ports (80/443 for Apache, 25 for SMTP).

#### Configure the `.env` File

Update your proxy's `.env` file to point directly to the Let's Encrypt certificates generated during the Apache/Certbot setup on your Core server:

```ini
# Domain configuration
DOMAIN=aurion.example.com  # Must match your Core Server domain

# TLS configuration pointing to Certbot paths
TLS_CERT=/etc/letsencrypt/live/aurion.example.com/fullchain.pem
TLS_KEY=/etc/letsencrypt/live/aurion.example.com/privkey.pem

```

#### Allow the Proxy to Read the Certificates

By default, Let's Encrypt directories are strictly restricted to `root`. Since the Aurion Proxy service runs under the `www-data` user, you must grant the `www-data` group read permissions to these folders, otherwise the proxy will fail to start:

```bash
# Give read and traverse permissions to the www-data group
sudo chown -R root:www-data /etc/letsencrypt/live/
sudo chown -R root:www-data /etc/letsencrypt/archive/
sudo chmod -R 750 /etc/letsencrypt/live/
sudo chmod -R 750 /etc/letsencrypt/archive/

```

### Step 3: Production Service (Systemd)

Since the proxy needs to bind to port 25 (a privileged port), it needs specific capabilities if you wish to run it under a non-root user (recommended for security).

Create the systemd service file `/etc/systemd/system/aurion-proxy.service`:

```ini
[Unit]
Description=Aurion Proxy Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/var/www/aurion-proxy
ExecStart=/var/www/aurion-proxy/aurion-proxy
Restart=always
RestartSec=5
EnvironmentFile=/var/www/aurion-proxy/.env

# Allow binding to port 25 without running as root
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target

```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable aurion-proxy
sudo systemctl start aurion-proxy

```

## 5. Logs and Verification

You can monitor the behavior of your proxy, including connection handling, routing API timeouts, and SMTP forwarding queues via `journalctl`:

```bash
# View real-time logs
sudo journalctl -u aurion-proxy -f

```