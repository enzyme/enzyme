# Self-Hosting Guide

This guide covers everything you need to deploy Enzyme on your own server.

## Requirements

- A Linux, macOS, or Windows server
- A domain name with an A record pointing to your server (required for automatic TLS)
- Ports 80 and 443 available for HTTPS (or 8080 for local/development use)

## Quick Start

1. Download the latest release for your platform from the [releases page](https://github.com/enzyme/enzyme/releases)
2. Start the server

```bash
# Download the binary (includes the web client)
curl -LO https://github.com/enzyme/enzyme/releases/latest/download/enzyme-linux-amd64
chmod +x enzyme-linux-amd64

# Start the server
./enzyme-linux-amd64
```

The server starts on `http://localhost:8080` and serves both the API and web client.

## Available Binaries

Each release includes pre-built binaries for six platforms. Each binary is a single self-contained file with the web client embedded:

| OS      | Architecture | Binary Name                |
| ------- | ------------ | -------------------------- |
| Linux   | x86_64       | `enzyme-linux-amd64`       |
| Linux   | ARM64        | `enzyme-linux-arm64`       |
| macOS   | x86_64       | `enzyme-darwin-amd64`      |
| macOS   | ARM64        | `enzyme-darwin-arm64`      |
| Windows | x86_64       | `enzyme-windows-amd64.exe` |
| Windows | ARM64        | `enzyme-windows-arm64.exe` |

## Data Directory

By default, Enzyme stores all data under `./data/` relative to where you run the binary:

```
data/
├── enzyme.db       # SQLite database
├── .signing_secret  # Auto-generated HMAC secret for file URLs
├── uploads/         # Uploaded files
└── certs/           # TLS certificates (if using auto TLS)
```

You can customize these paths via [configuration](./configuration.md). For production, use absolute paths:

```yaml
database:
  path: '/var/lib/enzyme/enzyme.db'

files:
  storage_path: '/var/lib/enzyme/uploads'
```

## Configuration

Create a `config.yaml` in the same directory as the binary (or pass `--config /path/to/config.yaml`). See the [Configuration Reference](./configuration.md) for all options.

A minimal production config with automatic TLS (Let's Encrypt):

```yaml
server:
  port: 443
  public_url: 'https://chat.example.com'
  allowed_origins: []
  tls:
    mode: 'auto'
    auto:
      domain: 'chat.example.com'
      email: 'admin@example.com'

email:
  enabled: true
  host: 'smtp.example.com'
  port: 587
  username: 'enzyme@example.com'
  password: 'your-smtp-password'
  from: 'enzyme@example.com'
```

When using auto TLS, Enzyme automatically redirects HTTP (port 80) to HTTPS (port 443).

Configuration can also be set via environment variables with the `ENZYME_` prefix or CLI flags. See the [Configuration Reference](./configuration.md) for details.

## Advanced: Reverse Proxy

Most deployments can use [built-in TLS](#built-in-tls) directly. A reverse proxy is an alternative if you prefer to handle TLS termination externally (e.g., you already run nginx or Caddy for other services).

### nginx

```nginx
server {
    listen 443 ssl http2;
    server_name chat.example.com;

    ssl_certificate     /etc/letsencrypt/live/chat.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/chat.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE support
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
    }
}
```

### Caddy

```
chat.example.com {
    reverse_proxy localhost:8080
}
```

When using a reverse proxy, set `allowed_origins` to an empty list since everything is same-origin:

```yaml
server:
  allowed_origins: []
```

## Built-in TLS

Enzyme has built-in TLS support if you don't want to use a reverse proxy.

### Automatic (Let's Encrypt)

```yaml
server:
  port: 443
  public_url: 'https://chat.example.com'
  tls:
    mode: 'auto'
    auto:
      domain: 'chat.example.com'
      email: 'admin@example.com'
      cache_dir: './data/certs'
```

This automatically obtains and renews certificates from Let's Encrypt. Requirements:

- Your domain's A record must point to the server's IP address
- Ports 80 and 443 must be reachable from the internet (port 80 is used for Let's Encrypt HTTP-01 challenges and HTTP-to-HTTPS redirect)

### Manual Certificates

```yaml
server:
  port: 443
  public_url: 'https://chat.example.com'
  tls:
    mode: 'manual'
    cert_file: '/etc/ssl/certs/chat.example.com.pem'
    key_file: '/etc/ssl/private/chat.example.com.key'
```

## Email Setup

Email is optional but enables password resets, workspace invites, and notification digests. Without email, users can only be invited via shareable invite links.

```yaml
email:
  enabled: true
  host: 'smtp.example.com'
  port: 587
  username: 'enzyme@example.com'
  password: 'your-smtp-password'
  from: 'Enzyme <enzyme@example.com>'
```

Enzyme works with any SMTP provider (Postmark, Mailgun, SendGrid, Amazon SES, self-hosted, etc.).

## Running as a systemd Service

Create `/etc/systemd/system/enzyme.service`:

```ini
[Unit]
Description=Enzyme
After=network.target

[Service]
Type=simple
User=enzyme
Group=enzyme
WorkingDirectory=/opt/enzyme
ExecStart=/opt/enzyme/enzyme --config /opt/enzyme/config.yaml
Restart=always
RestartSec=5

# Allow binding to port 443 without root
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/enzyme

[Install]
WantedBy=multi-user.target
```

Set it up:

```bash
# Create user
sudo useradd --system --shell /usr/sbin/nologin enzyme

# Create directories (include certs dir if using auto TLS)
sudo mkdir -p /opt/enzyme /var/lib/enzyme/uploads /var/lib/enzyme/certs
sudo chown -R enzyme:enzyme /opt/enzyme /var/lib/enzyme

# Copy files
sudo cp enzyme-linux-amd64 /opt/enzyme/enzyme
sudo cp config.yaml /opt/enzyme/config.yaml
sudo chmod +x /opt/enzyme/enzyme

# Enable and start
sudo systemctl enable enzyme
sudo systemctl start enzyme

# Check status
sudo systemctl status enzyme
sudo journalctl -u enzyme -f
```

## Scaling

The default configuration is tuned for a small server (2 GB RAM / 1 vCPU, ~100 users). If you're running on larger hardware or expect more users, see the [Scaling Guide](./scaling.md) for SQLite, HTTP, SSE, and OS-level tuning recommendations with example configs for small, medium, and large deployments.

## Telemetry (Optional)

Enzyme supports optional OpenTelemetry instrumentation for traces and metrics. When enabled, data is exported to any OTLP-compatible collector (Jaeger, Grafana Alloy, Datadog Agent, etc.). Disabled by default with zero overhead.

```bash
# Enable via environment variable
ENZYME_TELEMETRY_ENABLED=true ./enzyme

# Or in config.yaml
telemetry:
  enabled: true
  endpoint: "localhost:4317"
```

Enzyme captures application-level telemetry: HTTP request traces, database query spans, SSE connection metrics, and log correlation.

For **system-level metrics** (CPU, memory, disk, network), run an [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) alongside Enzyme with the `hostmetrics` receiver. The Collector acts as a local agent that scrapes host metrics and forwards everything (both Enzyme's app telemetry and system metrics) to your backend.

See the [Observability Guide](./observability.md) for full setup examples, what's captured, and sample Collector configs.

## Logs

Enzyme logs to stdout. Where logs end up depends on how you run the server:

- **Running directly** — logs appear in your terminal. Redirect to a file if needed: `./enzyme >> /var/log/enzyme.log 2>&1`
- **systemd service** — logs are captured by journald:

```bash
# Follow logs in real-time
sudo journalctl -u enzyme -f

# Show last 100 lines
sudo journalctl -u enzyme -n 100

# Show logs since last boot
sudo journalctl -u enzyme -b
```

- **Behind a reverse proxy** — Enzyme's own logs still go to stdout (or journald if using systemd). The reverse proxy (nginx, Caddy, etc.) has its own access logs separate from Enzyme's.

Configure log level and format in `config.yaml`:

```yaml
log:
  level: 'info' # debug, info, warn, error
  format: 'text' # text or json
```

Set `format: 'json'` for machine-parseable output (useful with log aggregation tools). Set `level: 'debug'` to see detailed output including per-request logs and email diagnostics. When running under systemd, log rotation is handled automatically by journald.

See the [Configuration Reference](./configuration.md#logging) for all logging options.

## Firewall

If your server runs a firewall (e.g., `ufw` on Ubuntu), make sure to open the required ports. **Always allow SSH first** to avoid locking yourself out:

```bash
sudo ufw allow 22/tcp     # SSH (do this FIRST)
sudo ufw allow 80/tcp     # HTTP (Let's Encrypt + redirect)
sudo ufw allow 443/tcp    # HTTPS
sudo ufw enable
```

## Backups

All persistent state is in the data directory (default: `./data/`):

1. **SQLite database** — the single `.db` file (default: `./data/enzyme.db`)
2. **Uploaded files** — the uploads directory (default: `./data/uploads/`)
3. **Signing secret** — `./data/.signing_secret` (used to sign file download URLs)

To back up:

```bash
# SQLite backup (safe to run while server is running — uses WAL mode)
sqlite3 /var/lib/enzyme/enzyme.db ".backup '/backups/enzyme-$(date +%Y%m%d).db'"

# File uploads and signing secret
rsync -a /var/lib/enzyme/uploads/ /backups/uploads/
cp /var/lib/enzyme/.signing_secret /backups/.signing_secret
```

## Upgrading

1. Download the new binary from the releases page
2. Stop the server (`sudo systemctl stop enzyme`)
3. Replace the binary
4. Start the server (`sudo systemctl start enzyme`)

Database migrations run automatically on startup. There is no manual migration step.

## Building from Source

```bash
git clone https://github.com/enzyme/enzyme.git
cd enzyme
make install
make build
```

The binary at `api/bin/enzyme` includes the embedded web client. Run it directly — no separate frontend serving needed.
