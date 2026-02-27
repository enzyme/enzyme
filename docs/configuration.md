# Configuration Reference

Enzyme is configured through a YAML config file, environment variables, or CLI flags. Settings are loaded in the following order, where later sources override earlier ones:

1. Built-in defaults
2. Config file (`config.yaml` or `--config <path>`)
3. Environment variables (`ENZYME_` prefix)
4. CLI flags

## Config File

Enzyme looks for `config.yaml` or `config.yml` in the current working directory. You can specify a custom path:

```bash
./enzyme --config /etc/enzyme/config.yaml
```

## Environment Variables

All settings can be set via environment variables with the `ENZYME_` prefix. Nested keys use underscores:

```bash
ENZYME_SERVER_PORT=9090
ENZYME_DATABASE_PATH=/var/lib/enzyme/enzyme.db
ENZYME_EMAIL_ENABLED=true
ENZYME_FILES_SIGNING_SECRET=your-secret-here
```

## CLI Flags

```bash
./enzyme --server.port 9090 --database.path /var/lib/enzyme/enzyme.db
```

Run `./enzyme --help` for all available flags.

---

## Logging

| Key          | Env Var             | CLI Flag       | Default | Description                                                        |
| ------------ | ------------------- | -------------- | ------- | ------------------------------------------------------------------ |
| `log.level`  | `ENZYME_LOG_LEVEL`  | `--log.level`  | `info`  | Minimum log level: `debug`, `info`, `warn`, `error`.               |
| `log.format` | `ENZYME_LOG_FORMAT` | `--log.format` | `text`  | Log output format: `text` (human-readable) or `json` (structured). |

## Server

| Key                      | Env Var                         | CLI Flag                   | Default                     | Description                                                                                                               |
| ------------------------ | ------------------------------- | -------------------------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `server.host`            | `ENZYME_SERVER_HOST`            | `--server.host`            | `0.0.0.0`                   | Address to bind the HTTP server to.                                                                                       |
| `server.port`            | `ENZYME_SERVER_PORT`            | `--server.port`            | `8080`                      | Port to listen on.                                                                                                        |
| `server.public_url`      | `ENZYME_SERVER_PUBLIC_URL`      | `--server.public_url`      | `http://localhost:8080`     | Public-facing URL. Used in emails (invite links, password resets). Must include the scheme.                               |
| `server.allowed_origins` | `ENZYME_SERVER_ALLOWED_ORIGINS` | `--server.allowed_origins` | `["http://localhost:3000"]` | CORS allowed origins. Set to `[]` for production (same-origin with embedded frontend). Each origin must include a scheme. |
| `server.read_timeout`    | `ENZYME_SERVER_READ_TIMEOUT`    |                            | `30s`                       | Max duration for reading the entire request (including body). Minimum: 1s.                                                |
| `server.write_timeout`   | `ENZYME_SERVER_WRITE_TIMEOUT`   |                            | `60s`                       | Max duration for writing the response. SSE connections override this per-connection. Minimum: 1s.                         |
| `server.idle_timeout`    | `ENZYME_SERVER_IDLE_TIMEOUT`    |                            | `120s`                      | Max duration to wait for the next request on a keep-alive connection. Minimum: 1s.                                        |

### TLS

| Key                         | Env Var                            | CLI Flag                      | Default        | Description                                                              |
| --------------------------- | ---------------------------------- | ----------------------------- | -------------- | ------------------------------------------------------------------------ |
| `server.tls.mode`           | `ENZYME_SERVER_TLS_MODE`           | `--server.tls.mode`           | `off`          | TLS mode: `off`, `auto` (Let's Encrypt), or `manual`.                    |
| `server.tls.cert_file`      | `ENZYME_SERVER_TLS_CERT_FILE`      | `--server.tls.cert_file`      |                | Path to TLS certificate file. Required for `manual` mode.                |
| `server.tls.key_file`       | `ENZYME_SERVER_TLS_KEY_FILE`       | `--server.tls.key_file`       |                | Path to TLS private key file. Required for `manual` mode.                |
| `server.tls.auto.domain`    | `ENZYME_SERVER_TLS_AUTO_DOMAIN`    | `--server.tls.auto.domain`    |                | Domain for automatic certificate provisioning. Required for `auto` mode. |
| `server.tls.auto.email`     | `ENZYME_SERVER_TLS_AUTO_EMAIL`     | `--server.tls.auto.email`     |                | Contact email for Let's Encrypt. Required for `auto` mode.               |
| `server.tls.auto.cache_dir` | `ENZYME_SERVER_TLS_AUTO_CACHE_DIR` | `--server.tls.auto.cache_dir` | `./data/certs` | Directory to cache TLS certificates.                                     |

## Database

| Key                       | Env Var                          | CLI Flag          | Default            | Description                                                                                              |
| ------------------------- | -------------------------------- | ----------------- | ------------------ | -------------------------------------------------------------------------------------------------------- |
| `database.path`           | `ENZYME_DATABASE_PATH`           | `--database.path` | `./data/enzyme.db` | Path to the SQLite database file. The directory must exist.                                              |
| `database.max_open_conns` | `ENZYME_DATABASE_MAX_OPEN_CONNS` |                   | `2`                | Max open database connections. Allows concurrent reads with WAL mode. Minimum: 1.                        |
| `database.busy_timeout`   | `ENZYME_DATABASE_BUSY_TIMEOUT`   |                   | `5000`             | Milliseconds to wait when the database is locked before returning SQLITE_BUSY. Minimum: 0.               |
| `database.cache_size`     | `ENZYME_DATABASE_CACHE_SIZE`     |                   | `-2000`            | SQLite page cache size. Negative values = KB (e.g., `-2000` = ~2 MB). Positive values = number of pages. |
| `database.mmap_size`      | `ENZYME_DATABASE_MMAP_SIZE`      |                   | `0`                | Memory-mapped I/O size in bytes. `0` disables mmap. Set higher for large databases on capable hardware.  |

Enzyme uses SQLite in WAL mode. No external database server is needed. See [Scaling Guide](scaling.md) for tuning guidance.

## Authentication

| Key                     | Env Var                        | CLI Flag                  | Default | Description                                                                                           |
| ----------------------- | ------------------------------ | ------------------------- | ------- | ----------------------------------------------------------------------------------------------------- |
| `auth.session_duration` | `ENZYME_AUTH_SESSION_DURATION` | `--auth.session_duration` | `720h`  | How long bearer tokens remain valid. Uses Go duration format (e.g., `720h` = 30 days, `24h`, `168h`). |
| `auth.bcrypt_cost`      | `ENZYME_AUTH_BCRYPT_COST`      |                           | `12`    | bcrypt hashing cost for passwords. Higher is more secure but slower. Range: 4-31.                     |

## File Storage

| Key                     | Env Var                        | CLI Flag                  | Default          | Description                                                                                                                                                                                                         |
| ----------------------- | ------------------------------ | ------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `files.storage_path`    | `ENZYME_FILES_STORAGE_PATH`    | `--files.storage_path`    | `./data/uploads` | Directory for uploaded files.                                                                                                                                                                                       |
| `files.max_upload_size` | `ENZYME_FILES_MAX_UPLOAD_SIZE` | `--files.max_upload_size` | `10485760`       | Maximum upload file size in bytes. Default is 10 MB.                                                                                                                                                                |
| `files.signing_secret`  | `ENZYME_FILES_SIGNING_SECRET`  |                           |                  | HMAC secret for signing file download URLs. If empty, a random secret is auto-generated and saved to `.signing_secret` in the database directory for persistence across restarts. You can also set this explicitly. |

## Email

Email is optional. When disabled, features like password reset and notification emails won't be available, but invite links will still work.

| Key              | Env Var                 | CLI Flag          | Default | Description                                                  |
| ---------------- | ----------------------- | ----------------- | ------- | ------------------------------------------------------------ |
| `email.enabled`  | `ENZYME_EMAIL_ENABLED`  | `--email.enabled` | `false` | Enable SMTP email sending.                                   |
| `email.host`     | `ENZYME_EMAIL_HOST`     |                   |         | SMTP server hostname.                                        |
| `email.port`     | `ENZYME_EMAIL_PORT`     |                   | `587`   | SMTP server port.                                            |
| `email.username` | `ENZYME_EMAIL_USERNAME` |                   |         | SMTP username.                                               |
| `email.password` | `ENZYME_EMAIL_PASSWORD` |                   |         | SMTP password.                                               |
| `email.from`     | `ENZYME_EMAIL_FROM`     |                   |         | Sender email address (e.g., `Enzyme <noreply@example.com>`). |

## Rate Limiting

Rate limiting protects authentication endpoints from brute-force attacks. Limits are per IP address.

| Key                                 | Env Var                                    | Default | Description                             |
| ----------------------------------- | ------------------------------------------ | ------- | --------------------------------------- |
| `rate_limit.enabled`                | `ENZYME_RATE_LIMIT_ENABLED`                | `true`  | Enable rate limiting.                   |
| `rate_limit.login.limit`            | `ENZYME_RATE_LIMIT_LOGIN_LIMIT`            | `10`    | Max login attempts per window.          |
| `rate_limit.login.window`           | `ENZYME_RATE_LIMIT_LOGIN_WINDOW`           | `1m`    | Login rate limit window.                |
| `rate_limit.register.limit`         | `ENZYME_RATE_LIMIT_REGISTER_LIMIT`         | `5`     | Max registration attempts per window.   |
| `rate_limit.register.window`        | `ENZYME_RATE_LIMIT_REGISTER_WINDOW`        | `1h`    | Registration rate limit window.         |
| `rate_limit.forgot_password.limit`  | `ENZYME_RATE_LIMIT_FORGOT_PASSWORD_LIMIT`  | `5`     | Max password reset requests per window. |
| `rate_limit.forgot_password.window` | `ENZYME_RATE_LIMIT_FORGOT_PASSWORD_WINDOW` | `15m`   | Password reset request window.          |
| `rate_limit.reset_password.limit`   | `ENZYME_RATE_LIMIT_RESET_PASSWORD_LIMIT`   | `10`    | Max password reset attempts per window. |
| `rate_limit.reset_password.window`  | `ENZYME_RATE_LIMIT_RESET_PASSWORD_WINDOW`  | `15m`   | Password reset attempt window.          |

## SSE (Real-Time Events)

| Key                      | Env Var                         | Default | Description                                                                            |
| ------------------------ | ------------------------------- | ------- | -------------------------------------------------------------------------------------- |
| `sse.event_retention`    | `ENZYME_SSE_EVENT_RETENTION`    | `24h`   | How long SSE events are stored for reconnection catch-up.                              |
| `sse.cleanup_interval`   | `ENZYME_SSE_CLEANUP_INTERVAL`   | `1h`    | How often old SSE events are purged from the database.                                 |
| `sse.heartbeat_interval` | `ENZYME_SSE_HEARTBEAT_INTERVAL` | `30s`   | How often heartbeat events are sent to keep SSE connections alive. Minimum: 5s.        |
| `sse.client_buffer_size` | `ENZYME_SSE_CLIENT_BUFFER_SIZE` | `256`   | Channel buffer size per SSE client. Increase for high-traffic workspaces. Minimum: 16. |

## Telemetry (OpenTelemetry)

Optional observability via OpenTelemetry. When enabled, Enzyme exports traces and metrics to any OTLP-compatible collector (Jaeger, Grafana Alloy, Datadog Agent, etc.). Disabled by default with zero overhead.

| Key                      | Env Var                         | CLI Flag                   | Default          | Description                                                                    |
| ------------------------ | ------------------------------- | -------------------------- | ---------------- | ------------------------------------------------------------------------------ |
| `telemetry.enabled`      | `ENZYME_TELEMETRY_ENABLED`      | `--telemetry.enabled`      | `false`          | Enable OpenTelemetry instrumentation.                                          |
| `telemetry.endpoint`     | `ENZYME_TELEMETRY_ENDPOINT`     | `--telemetry.endpoint`     | `localhost:4317` | OTLP collector endpoint (host:port).                                           |
| `telemetry.protocol`     | `ENZYME_TELEMETRY_PROTOCOL`     | `--telemetry.protocol`     | `grpc`           | Export protocol: `grpc` or `http`.                                             |
| `telemetry.insecure`     | `ENZYME_TELEMETRY_INSECURE`     | `--telemetry.insecure`     | `true`           | Use plaintext (no TLS) for OTLP export.                                        |
| `telemetry.sample_rate`  | `ENZYME_TELEMETRY_SAMPLE_RATE`  | `--telemetry.sample_rate`  | `1.0`            | Trace sampling rate. `1.0` = sample everything, `0.1` = sample 10%.            |
| `telemetry.service_name` | `ENZYME_TELEMETRY_SERVICE_NAME` | `--telemetry.service_name` | `enzyme`         | Service name reported to the collector. Useful for multi-instance deployments. |

See the [Observability Guide](observability.md) for details on what's captured (traces, metrics, log correlation) and setup examples.

### Example: Local development with Jaeger

```bash
# Start Jaeger all-in-one
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest

# Start Enzyme with telemetry
ENZYME_TELEMETRY_ENABLED=true ./enzyme
```

Then open `http://localhost:16686` to view traces.

## Full Example

```yaml
log:
  level: 'info'
  format: 'text'

server:
  host: '0.0.0.0'
  port: 443
  public_url: 'https://chat.example.com'
  allowed_origins: [] # Same-origin (embedded frontend)
  tls:
    mode: 'auto'
    auto:
      domain: 'chat.example.com'
      email: 'admin@example.com'
      cache_dir: './data/certs'
  read_timeout: '30s'
  write_timeout: '60s'
  idle_timeout: '120s'

database:
  path: '/var/lib/enzyme/enzyme.db'
  max_open_conns: 2
  busy_timeout: 5000
  cache_size: -2000
  mmap_size: 0

auth:
  session_duration: '720h'
  bcrypt_cost: 12

files:
  storage_path: '/var/lib/enzyme/uploads'
  max_upload_size: 26214400 # 25 MB
  signing_secret: 'your-random-secret-here'

email:
  enabled: true
  host: 'smtp.postmarkapp.com'
  port: 587
  username: 'your-api-key'
  password: 'your-api-key'
  from: 'Enzyme <notifications@example.com>'

rate_limit:
  enabled: true
  login:
    limit: 10
    window: '1m'
  register:
    limit: 5
    window: '1h'
  forgot_password:
    limit: 5
    window: '15m'
  reset_password:
    limit: 10
    window: '15m'

sse:
  event_retention: '24h'
  cleanup_interval: '1h'
  heartbeat_interval: '30s'
  client_buffer_size: 256

telemetry:
  enabled: false
  endpoint: 'localhost:4317'
  protocol: 'grpc'
  sample_rate: 1.0
  service_name: 'enzyme'
```
