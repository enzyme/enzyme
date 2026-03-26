# Enzyme Server

Go backend for Enzyme, a self-hostable Slack alternative.

> Part of the [Enzyme monorepo](../README.md). See root for full setup.

## Features

- **Workspaces** - Create teams with role-based permissions (owner, admin, member, guest)
- **Channels** - Public, private, DM, and group DM conversations
- **Messages** - Rich messaging with flat threading (Slack-style) and emoji reactions
- **Real-time** - Server-Sent Events (SSE) with automatic reconnection and catch-up
- **Presence** - Online/away/offline status with automatic away detection
- **File Uploads** - Multipart uploads stored to disk
- **Email** - Optional SMTP integration with graceful degradation

## Quick Start

```bash
# Build and run
make dev

# Or build binary
make build
./bin/enzyme
```

The server starts on `http://localhost:8080` with zero configuration required.

## Configuration

Configuration is loaded in layers: defaults < config file < environment variables < CLI flags.

### Config File

Create `config.yaml` (see `config.example.yaml`):

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  public_url: "https://chat.example.com"

database:
  path: "./data/enzyme.db"

auth:
  session_duration: "720h"  # 30 days
  bcrypt_cost: 12

files:
  storage_path: "./data/uploads"
  max_upload_size: 10485760  # 10MB

email:
  enabled: true
  host: "smtp.example.com"
  port: 587
  username: ""
  password: ""
  from: "noreply@example.com"
```

### Environment Variables

All config options can be set via environment variables with `ENZYME_` prefix:

```bash
ENZYME_SERVER_PORT=3000
ENZYME_DATABASE_PATH=/var/lib/enzyme/enzyme.db
```

### CLI Flags

```bash
./enzyme --server.port=3000 --database.path=/data/enzyme.db
```

## API Endpoints

All API endpoints are under `/api/`. Protected endpoints require `Authorization: Bearer <token>` header.

### Authentication
```
POST /api/auth/register        # Create account (auto-login)
POST /api/auth/login           # Email + password login
POST /api/auth/logout          # Clear session
POST /api/auth/forgot-password # Request password reset
POST /api/auth/reset-password  # Reset with token
GET  /api/auth/me              # Current user + workspaces
```

### Workspaces
```
POST /api/workspaces/create
POST /api/workspaces/{id}/update
GET  /api/workspaces/{id}
POST /api/workspaces/{id}/members/list
POST /api/workspaces/{id}/members/remove
POST /api/workspaces/{id}/members/update-role
POST /api/workspaces/{id}/invites/create
POST /api/invites/{code}/accept
```

### Channels
```
POST /api/workspaces/{id}/channels/create
POST /api/workspaces/{id}/channels/list
POST /api/workspaces/{id}/channels/dm
POST /api/channels/{id}/update
POST /api/channels/{id}/archive
POST /api/channels/{id}/members/add
POST /api/channels/{id}/members/list
POST /api/channels/{id}/join
POST /api/channels/{id}/leave
```

### Messages
```
POST /api/channels/{id}/messages/send
POST /api/channels/{id}/messages/list
POST /api/messages/{id}/update
POST /api/messages/{id}/delete
POST /api/messages/{id}/reactions/add
POST /api/messages/{id}/reactions/remove
POST /api/messages/{id}/thread/list
```

### Files
```
POST /api/channels/{id}/files/upload  # Multipart form
GET  /api/files/{id}/download
POST /api/files/{id}/delete
```

### Real-time Events
```
GET  /api/workspaces/{id}/events      # SSE stream
POST /api/workspaces/{id}/typing/start
POST /api/workspaces/{id}/typing/stop
```

### Error Format
```json
{
  "error": {
    "code": "NOT_CHANNEL_MEMBER",
    "message": "You are not a member of this channel"
  }
}
```

## SSE Events

Connect to `/api/workspaces/{id}/events` with `Authorization: Bearer <token>` header for real-time updates. Supports `Last-Event-ID` header for reconnection catch-up.

Event types:
- `connected`, `heartbeat`
- `message.new`, `message.updated`, `message.deleted`
- `message.pinned`, `message.unpinned`
- `reaction.added`, `reaction.removed`
- `channel.created`, `channel.updated`, `channel.archived`
- `channel.member_added`, `channel.member_removed`
- `channel.read`, `channels.invalidate`
- `typing.start`, `typing.stop`
- `presence.changed`, `presence.initial`
- `notification`
- `emoji.created`, `emoji.deleted`
- `member.banned`, `member.unbanned`, `member.left`, `member.role_changed`
- `workspace.updated`
- `scheduled_message.created`, `scheduled_message.updated`, `scheduled_message.deleted`, `scheduled_message.sent`, `scheduled_message.failed`

## Project Structure

```
server/
├── cmd/enzyme/main.go           # Entry point, graceful shutdown
├── internal/
│   ├── app/app.go                # Dependency wiring, startup
│   ├── config/                   # Layered configuration
│   ├── database/                 # SQLite connection, migrations
│   ├── auth/                     # Authentication, sessions
│   ├── user/                     # User model, repository
│   ├── workspace/                # Workspaces, memberships, invites
│   ├── channel/                  # Channels, DMs
│   ├── message/                  # Messages, reactions, threading
│   ├── file/                     # File uploads, storage
│   ├── sse/                      # SSE hub, broadcasting
│   ├── presence/                 # Online status tracking
│   ├── email/                    # SMTP sender, templates
│   └── server/                   # HTTP server, router
├── config.example.yaml
├── Makefile
└── go.mod
```

## Technology Stack

| Component | Choice |
|-----------|--------|
| Language | Go |
| Database | SQLite (modernc.org/sqlite - pure Go) |
| Migrations | pressly/goose |
| Config | knadh/koanf |
| Auth | Bearer token sessions |
| Password | bcrypt (cost 12) |
| IDs | ULID |
| Router | go-chi/chi |

## Development

```bash
# Run in development mode
make dev

# Seed database with sample data (8 users, 2 workspaces, ~4k messages)
# Login as alice@example.com / password (or any user — all use "password")
make seed

# Build binary
make build

# Run tests
make test

# Lint
make lint

# Tidy dependencies
make deps
```

## Database

SQLite with WAL mode for concurrent reads. Single connection pool to avoid SQLITE_BUSY errors.

Migrations are embedded and run automatically on startup. Database file is created at the configured path (default: `./data/enzyme.db`).

## Authorization Model

**Workspace roles** (cannot be overridden per-user):
- `owner` - Full access, cannot be removed
- `admin` - Most permissions except workspace deletion
- `member` - Create channels, post messages
- `guest` - Explicit channel access only

**Channel roles** (optional per-user override):
- `admin` - Manage channel settings
- `poster` - Post messages (default)
- `viewer` - Read-only

## License

[Add license here]
