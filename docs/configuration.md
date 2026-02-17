# Configuration Reference

Feather is configured through a YAML config file, environment variables, or CLI flags. Settings are loaded in the following order, where later sources override earlier ones:

1. Built-in defaults
2. Config file (`config.yaml` or `--config <path>`)
3. Environment variables (`FEATHER_` prefix)
4. CLI flags

## Config File

Feather looks for `config.yaml` or `config.yml` in the current working directory. You can specify a custom path:

```bash
./feather --config /etc/feather/config.yaml
```

## Environment Variables

All settings can be set via environment variables with the `FEATHER_` prefix. Nested keys use underscores:

```bash
FEATHER_SERVER_PORT=9090
FEATHER_DATABASE_PATH=/var/lib/feather/feather.db
FEATHER_EMAIL_ENABLED=true
FEATHER_FILES_SIGNING_SECRET=your-secret-here
```

## CLI Flags

```bash
./feather --server.port 9090 --database.path /var/lib/feather/feather.db
```

Run `./feather --help` for all available flags.

---

## Logging

| Key          | Env Var              | CLI Flag       | Default | Description                                                        |
| ------------ | -------------------- | -------------- | ------- | ------------------------------------------------------------------ |
| `log.level`  | `FEATHER_LOG_LEVEL`  | `--log.level`  | `info`  | Minimum log level: `debug`, `info`, `warn`, `error`.               |
| `log.format` | `FEATHER_LOG_FORMAT` | `--log.format` | `text`  | Log output format: `text` (human-readable) or `json` (structured). |

## Server

| Key                      | Env Var                          | CLI Flag                   | Default                     | Description                                                                                                               |
| ------------------------ | -------------------------------- | -------------------------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `server.host`            | `FEATHER_SERVER_HOST`            | `--server.host`            | `0.0.0.0`                   | Address to bind the HTTP server to.                                                                                       |
| `server.port`            | `FEATHER_SERVER_PORT`            | `--server.port`            | `8080`                      | Port to listen on.                                                                                                        |
| `server.public_url`      | `FEATHER_SERVER_PUBLIC_URL`      | `--server.public_url`      | `http://localhost:8080`     | Public-facing URL. Used in emails (invite links, password resets). Must include the scheme.                               |
| `server.allowed_origins` | `FEATHER_SERVER_ALLOWED_ORIGINS` | `--server.allowed_origins` | `["http://localhost:3000"]` | CORS allowed origins. Set to `[]` for production (same-origin with embedded frontend). Each origin must include a scheme. |

### TLS

| Key                         | Env Var                             | CLI Flag                      | Default        | Description                                                              |
| --------------------------- | ----------------------------------- | ----------------------------- | -------------- | ------------------------------------------------------------------------ |
| `server.tls.mode`           | `FEATHER_SERVER_TLS_MODE`           | `--server.tls.mode`           | `off`          | TLS mode: `off`, `auto` (Let's Encrypt), or `manual`.                    |
| `server.tls.cert_file`      | `FEATHER_SERVER_TLS_CERT_FILE`      | `--server.tls.cert_file`      |                | Path to TLS certificate file. Required for `manual` mode.                |
| `server.tls.key_file`       | `FEATHER_SERVER_TLS_KEY_FILE`       | `--server.tls.key_file`       |                | Path to TLS private key file. Required for `manual` mode.                |
| `server.tls.auto.domain`    | `FEATHER_SERVER_TLS_AUTO_DOMAIN`    | `--server.tls.auto.domain`    |                | Domain for automatic certificate provisioning. Required for `auto` mode. |
| `server.tls.auto.email`     | `FEATHER_SERVER_TLS_AUTO_EMAIL`     | `--server.tls.auto.email`     |                | Contact email for Let's Encrypt. Required for `auto` mode.               |
| `server.tls.auto.cache_dir` | `FEATHER_SERVER_TLS_AUTO_CACHE_DIR` | `--server.tls.auto.cache_dir` | `./data/certs` | Directory to cache TLS certificates.                                     |

## Database

| Key             | Env Var                 | CLI Flag          | Default             | Description                                                 |
| --------------- | ----------------------- | ----------------- | ------------------- | ----------------------------------------------------------- |
| `database.path` | `FEATHER_DATABASE_PATH` | `--database.path` | `./data/feather.db` | Path to the SQLite database file. The directory must exist. |

Feather uses SQLite in WAL mode with a single connection. No external database server is needed.

## Authentication

| Key                     | Env Var                         | CLI Flag                  | Default | Description                                                                                           |
| ----------------------- | ------------------------------- | ------------------------- | ------- | ----------------------------------------------------------------------------------------------------- |
| `auth.session_duration` | `FEATHER_AUTH_SESSION_DURATION` | `--auth.session_duration` | `720h`  | How long bearer tokens remain valid. Uses Go duration format (e.g., `720h` = 30 days, `24h`, `168h`). |
| `auth.bcrypt_cost`      | `FEATHER_AUTH_BCRYPT_COST`      |                           | `12`    | bcrypt hashing cost for passwords. Higher is more secure but slower. Range: 4-31.                     |

## File Storage

| Key                     | Env Var                         | CLI Flag                  | Default          | Description                                                                                                                                                                                                         |
| ----------------------- | ------------------------------- | ------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `files.storage_path`    | `FEATHER_FILES_STORAGE_PATH`    | `--files.storage_path`    | `./data/uploads` | Directory for uploaded files.                                                                                                                                                                                       |
| `files.max_upload_size` | `FEATHER_FILES_MAX_UPLOAD_SIZE` | `--files.max_upload_size` | `10485760`       | Maximum upload file size in bytes. Default is 10 MB.                                                                                                                                                                |
| `files.signing_secret`  | `FEATHER_FILES_SIGNING_SECRET`  |                           |                  | HMAC secret for signing file download URLs. If empty, a random secret is auto-generated and saved to `.signing_secret` in the database directory for persistence across restarts. You can also set this explicitly. |

## Email

Email is optional. When disabled, features like password reset and notification emails won't be available, but invite links will still work.

| Key              | Env Var                  | CLI Flag          | Default | Description                                                   |
| ---------------- | ------------------------ | ----------------- | ------- | ------------------------------------------------------------- |
| `email.enabled`  | `FEATHER_EMAIL_ENABLED`  | `--email.enabled` | `false` | Enable SMTP email sending.                                    |
| `email.host`     | `FEATHER_EMAIL_HOST`     |                   |         | SMTP server hostname.                                         |
| `email.port`     | `FEATHER_EMAIL_PORT`     |                   | `587`   | SMTP server port.                                             |
| `email.username` | `FEATHER_EMAIL_USERNAME` |                   |         | SMTP username.                                                |
| `email.password` | `FEATHER_EMAIL_PASSWORD` |                   |         | SMTP password.                                                |
| `email.from`     | `FEATHER_EMAIL_FROM`     |                   |         | Sender email address (e.g., `Feather <noreply@example.com>`). |

## Rate Limiting

Rate limiting protects authentication endpoints from brute-force attacks. Limits are per IP address.

| Key                                 | Env Var                                     | Default | Description                             |
| ----------------------------------- | ------------------------------------------- | ------- | --------------------------------------- |
| `rate_limit.enabled`                | `FEATHER_RATE_LIMIT_ENABLED`                | `true`  | Enable rate limiting.                   |
| `rate_limit.login.limit`            | `FEATHER_RATE_LIMIT_LOGIN_LIMIT`            | `10`    | Max login attempts per window.          |
| `rate_limit.login.window`           | `FEATHER_RATE_LIMIT_LOGIN_WINDOW`           | `1m`    | Login rate limit window.                |
| `rate_limit.register.limit`         | `FEATHER_RATE_LIMIT_REGISTER_LIMIT`         | `5`     | Max registration attempts per window.   |
| `rate_limit.register.window`        | `FEATHER_RATE_LIMIT_REGISTER_WINDOW`        | `1h`    | Registration rate limit window.         |
| `rate_limit.forgot_password.limit`  | `FEATHER_RATE_LIMIT_FORGOT_PASSWORD_LIMIT`  | `5`     | Max password reset requests per window. |
| `rate_limit.forgot_password.window` | `FEATHER_RATE_LIMIT_FORGOT_PASSWORD_WINDOW` | `15m`   | Password reset request window.          |
| `rate_limit.reset_password.limit`   | `FEATHER_RATE_LIMIT_RESET_PASSWORD_LIMIT`   | `10`    | Max password reset attempts per window. |
| `rate_limit.reset_password.window`  | `FEATHER_RATE_LIMIT_RESET_PASSWORD_WINDOW`  | `15m`   | Password reset attempt window.          |

## SSE (Real-Time Events)

| Key                    | Env Var                        | Default | Description                                               |
| ---------------------- | ------------------------------ | ------- | --------------------------------------------------------- |
| `sse.event_retention`  | `FEATHER_SSE_EVENT_RETENTION`  | `24h`   | How long SSE events are stored for reconnection catch-up. |
| `sse.cleanup_interval` | `FEATHER_SSE_CLEANUP_INTERVAL` | `1h`    | How often old SSE events are purged from the database.    |

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

database:
  path: '/var/lib/feather/feather.db'

auth:
  session_duration: '720h'
  bcrypt_cost: 12

files:
  storage_path: '/var/lib/feather/uploads'
  max_upload_size: 26214400 # 25 MB
  signing_secret: 'your-random-secret-here'

email:
  enabled: true
  host: 'smtp.postmarkapp.com'
  port: 587
  username: 'your-api-key'
  password: 'your-api-key'
  from: 'Feather <notifications@example.com>'

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
```
