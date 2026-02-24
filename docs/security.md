# Security

Enzyme is designed to be self-hosted: the operator owns the infrastructure, the database, and the files. The security model assumes trust in the server operator and focuses on protecting data in transit, enforcing access control between users, and minimizing the attack surface of the application itself. This document covers what Enzyme does, what it intentionally does not do, and why.

## Transport Security (TLS)

Enzyme supports three TLS modes, configured via `tls.mode`:

| Mode     | Description                                                            |
| -------- | ---------------------------------------------------------------------- |
| `off`    | Plain HTTP (default). Suitable behind a TLS-terminating reverse proxy. |
| `auto`   | Automatic certificates via Let's Encrypt (ACME HTTP-01 challenge).     |
| `manual` | Operator-provided certificate and key files.                           |

In `auto` mode:

- TLS 1.2 is the minimum enforced version.
- An HTTP-to-HTTPS redirect server runs on port 80.
- The certificate cache directory is created with `0700` permissions.
- Only hostnames matching the configured allowlist are issued certificates (`autocert.HostWhitelist`).

In `manual` mode, Go's default TLS configuration applies (TLS 1.2 minimum in Go 1.18+).

Enzyme does not set security response headers (HSTS, CSP, X-Frame-Options, X-Content-Type-Options). If you need these, add them via a reverse proxy. See [self-hosting.md](self-hosting.md) for TLS setup and [configuration.md](configuration.md) for all TLS options.

## Authentication

### Bearer Tokens

API authentication uses `Authorization: Bearer <token>` headers. Tokens are 32 bytes generated from `crypto/rand`, hex-encoded to a 64-character string. Only the SHA-256 hash of the token is stored in the database — a database leak does not expose usable tokens.

Tokens expire after 30 days by default (configurable via `auth.session_duration`). Expired sessions are cleaned up by a background goroutine every hour.

### Why Headers, Not Cookies

Using the `Authorization` header instead of cookies eliminates CSRF as a vulnerability class entirely and provides a single authentication mechanism for both API requests and SSE connections. The tradeoff is that the token is stored in `localStorage`, which is accessible to JavaScript running on the same origin. This is an acceptable tradeoff for a self-hosted application where the operator controls the origin.

### Passwords

Passwords are hashed with bcrypt at cost 12 (configurable via `auth.bcrypt_cost`, minimum 10). A minimum length of 8 characters is enforced on both registration and password reset.

### Password Reset

Reset tokens are 32 bytes from `crypto/rand` (hex-encoded), expire after 1 hour, and are single-use. The forgot-password endpoint returns a success response regardless of whether the email exists, preventing email enumeration.

### Rate Limiting

Per-IP rate limiting is enabled by default on all authentication endpoints:

| Endpoint        | Limit       | Window     |
| --------------- | ----------- | ---------- |
| Login           | 10 requests | 1 minute   |
| Register        | 5 requests  | 1 hour     |
| Forgot password | 5 requests  | 15 minutes |
| Reset password  | 10 requests | 15 minutes |

When a limit is exceeded, the server returns HTTP 429 with a `Retry-After` header. Rate limits are configurable — see [configuration.md](configuration.md).

Note: rate limiting uses the remote IP address after applying `X-Forwarded-For` / `X-Real-IP` headers. If Enzyme is behind a reverse proxy, ensure the proxy is configured to set these headers correctly.

## Authorization

Authorization is role-based and scoped to individual workspaces. There are no server-level admin roles — each workspace has its own owner, admins, members, and guests. All data access (messages, files, channels, SSE events) is scoped to workspace and channel membership.

See [permissions.md](permissions.md) for the full RBAC permission matrix.

## File Security and Signed URLs

Files are not served directly. To access a file, the client first requests a signed URL from an authenticated endpoint. The server generates the URL by computing an HMAC-SHA256 signature over `fileID:userID:expiresUnix` using a server-side signing secret, with constant-time comparison on verification.

Signed URLs expire after 1 hour.

**Why signed URLs?** HTML `<img>` tags and other embedded resources cannot send `Authorization` headers. Signed URLs embed authentication into the URL itself — the same pattern used by S3 pre-signed URLs.

### Signing Secret

The signing secret is 32 bytes from `crypto/rand` (hex-encoded). On first startup, it is auto-generated and persisted to a `.signing_secret` file alongside the database with `0600` permissions (owner read/write only). The secret can also be set explicitly via `files.signing_secret` in the config file.

### Upload Protections

- **Filename sanitization**: `filepath.Base` strips directory components; forward slashes, backslashes, and null bytes are removed; filenames are truncated to 255 characters. Files are stored on disk using a generated ULID, not the user-supplied name.
- **Size limits**: 10 MB for file uploads by default (configurable via [`files.max_upload_size`](configuration.md#file-storage)), 5 MB for avatars and workspace icons, 256 KB for custom emoji.

### Download Access Control

Even with a valid signed URL, the server verifies channel membership before serving a file. A signed URL alone is not sufficient — the requesting user must still have access to the channel where the file was posted.

## Data Storage

Enzyme uses SQLite with parameterized queries throughout — all user input is passed as bind parameters, never interpolated into SQL strings.

FTS5 full-text search queries are sanitized before execution: each word is individually quoted after stripping any double-quote characters, preventing FTS5 operator injection. The sanitized query is then passed as a bind parameter.

There is no encryption at rest at the application level. This is intentional — see [What Enzyme Does Not Do](#what-enzyme-does-not-do-and-why) below.

## Real-Time Events (SSE)

SSE connections require an `Authorization: Bearer` header. There is no query-parameter fallback for token passing.

Before a connection is established, the server verifies that the authenticated user is a member of the requested workspace. Events are broadcast only to members of the relevant channel.

On reconnection, the client sends a `Last-Event-ID` header and the server replays any missed events from the last 24 hours (configurable via `sse.event_retention`). Events older than the retention window are periodically cleaned up.

## Cross-Origin Requests (CORS)

CORS is whitelist-based and only active when `server.allowed_origins` is non-empty. When the list is empty (the default for production same-origin deployments), no CORS headers are sent — this is the most restrictive configuration.

Each origin in the list must include a scheme (`http://` or `https://`). Only the methods `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, and `OPTIONS` are allowed. Only `Content-Type` and `Authorization` headers are permitted. Credentials are not included.

The default configuration (`["http://localhost:3000"]`) is intended for local development only.

## Desktop Client (Electron)

The Electron desktop client is configured with all security defaults at their strictest:

- `contextIsolation: true` — the renderer cannot access Node.js APIs.
- `nodeIntegration: false` — Node.js is not available in the renderer process.
- `sandbox: true` — the renderer process is OS-level sandboxed.

The preload bridge exposes exactly two values via `contextBridge`: `platform` (a string) and `isElectron` (a boolean). There are no IPC channels, no filesystem access, and no shell command execution from the renderer.

## Frontend (XSS Prevention)

- Message content is stored and rendered as plain text through React's JSX rendering, which escapes HTML entities by default.
- There are no uses of `dangerouslySetInnerHTML` for user-supplied content anywhere in the frontend.
- Rich text editing uses a structured editor (TipTap/ProseMirror), not raw HTML insertion.

## What Enzyme Does Not Do (and Why)

### End-to-end encryption

Would prevent full-text search, shared message history for new channel members, reconnection catch-up, and notification digests. In a self-hosted model, you already trust the server. Use TLS for data in transit and disk-level encryption for data at rest.

### Encryption at rest

Better handled at the infrastructure level (LUKS, dm-crypt, cloud provider encryption). Application-level encryption adds key management complexity without meaningful benefit when the operator controls the server. Enable disk-level or filesystem-level encryption instead.

### Security headers (HSTS, CSP, etc.)

Enzyme is a single binary serving an SPA. A strict CSP could break custom deployments, and HSTS requires careful consideration of TLS configuration. Add headers via a reverse proxy (nginx, Caddy, Traefik).

### Token rotation / refresh tokens

Adds complexity. Tokens are valid for 30 days and are revocable via logout (deletes the session server-side). Reduce `auth.session_duration` for shorter-lived tokens if needed.

### Multi-factor authentication

Not yet implemented. Place Enzyme behind a reverse proxy with MFA support (Authelia, Authentik, Keycloak).

### Registration restriction

There is no server-level admin role to gate signups. Restrict network access to the server via firewall rules or a VPN.

## Reporting Vulnerabilities

If you discover a security vulnerability in Enzyme, please report it responsibly by opening a security advisory on the [GitHub repository](https://github.com/anthropics/enzyme). Do not open a public issue for security vulnerabilities.

Include a description of the vulnerability, steps to reproduce, and the potential impact. We will acknowledge receipt and work on a fix promptly.
