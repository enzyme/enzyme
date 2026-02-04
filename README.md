# Feather

A self-hostable Slack alternative built with Go and React.

## Features

- **Workspaces** - Create teams with role-based permissions (owner, admin, member, guest)
- **Channels** - Public, private, DM, and group DM conversations
- **Messages** - Rich messaging with flat threading (Slack-style) and emoji reactions
- **Real-time** - Server-Sent Events (SSE) with automatic reconnection
- **Presence** - Online/away/offline status with typing indicators
- **File Uploads** - Multipart uploads with configurable storage

## Quick Start

```bash
# Install dependencies
make install

# Start development servers
make dev
```

This starts:
- **API** at http://localhost:8080
- **Web** at http://localhost:3000

## Project Structure

```
feather/
├── api/                    # Go backend server
├── clients/web/            # React frontend
├── packages/api-client/    # Shared TypeScript types
└── Makefile                # Build orchestration
```

## Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start API and web dev servers |
| `make build` | Build all packages |
| `make test` | Run all tests |
| `make generate-types` | Regenerate types from OpenAPI |
| `make lint` | Lint all code |

## Type Sharing

Types are generated from `api/openapi.yaml` and shared via the `@feather/api-client` package:

```bash
# Regenerate after API changes
make generate-types
```

## CI/CD

### Continuous Integration

CI runs automatically on every push to `main` and on pull requests:

- Linting (Go vet, ESLint)
- Tests (Go tests)
- Type checking (TypeScript)
- Build verification

### Releases

Releases are triggered manually by pushing a version tag:

```bash
# Create and push a release tag
git tag v1.0.0
git push origin v1.0.0
```

This builds binaries for 6 platforms and creates a GitHub Release with:
- `feather-linux-amd64`, `feather-linux-arm64`
- `feather-darwin-amd64`, `feather-darwin-arm64`
- `feather-windows-amd64.exe`, `feather-windows-arm64.exe`
- `feather-web.tar.gz` (frontend bundle)

## Documentation

| Package | README |
|---------|--------|
| API (Go) | [api/README.md](api/README.md) |
| Web (React) | [clients/web/README.md](clients/web/README.md) |

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go, Chi, SQLite |
| Frontend | React, TypeScript, Vite, TanStack Query, Zustand, Tailwind |
| UI Components | React Aria Components, tailwind-variants |
| Real-time | Server-Sent Events |
| Types | OpenAPI 3.0, oapi-codegen, openapi-typescript |

## License

[Add license here]
