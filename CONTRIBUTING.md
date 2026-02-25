# Contributing to Enzyme

Thanks for your interest in contributing to Enzyme! This guide covers everything you need to get started.

## Development Setup

### Prerequisites

- [Go](https://go.dev/) 1.24+
- [Node.js](https://nodejs.org/) 22+
- [pnpm](https://pnpm.io/) 9+

### Getting Started

```bash
# Clone the repo
git clone https://github.com/enzyme/enzyme.git
cd enzyme

# Install dependencies
make install

# Start development servers (API + Web)
make dev
```

This starts:

- **API** at http://localhost:8080
- **Web** at http://localhost:3000

### Seed Data

Run `make seed` to populate the database with sample users, workspaces, channels, and messages for development. Login with any of the seed users (e.g. `alice@example.com` / `password`).

## Project Structure

```
enzyme/
├── api/                        # Go backend server
│   ├── openapi.yaml            # API specification (source of truth)
│   ├── cmd/enzyme/main.go      # Entry point
│   └── internal/
│       ├── handler/            # HTTP handler implementations
│       ├── database/           # SQLite connection, migrations
│       ├── auth/               # Authentication (sessions, middleware)
│       ├── user/               # User domain (model, repository)
│       ├── workspace/          # Workspace domain
│       ├── channel/            # Channel domain
│       ├── message/            # Message domain
│       ├── file/               # File uploads
│       ├── sse/                # Server-Sent Events (real-time)
│       └── presence/           # Online/away/offline tracking
├── apps/
│   ├── desktop/                # Electron desktop client
│   ├── web/                    # React frontend
│   └── website/                # Marketing site (Eleventy)
├── packages/
│   └── api-client/             # Shared TypeScript types (generated from OpenAPI)
├── docs/                       # Documentation
└── Makefile                    # Build orchestration
```

## Build Commands

| Command               | Description                    |
| --------------------- | ------------------------------ |
| `make install`        | Install all dependencies       |
| `make dev`            | Start API and web dev servers  |
| `make dev DESKTOP=1`  | Also start Electron            |
| `make build`          | Build all packages             |
| `make test`           | Run all tests                  |
| `make generate-types` | Regenerate types from OpenAPI  |
| `make seed`           | Seed database with sample data |
| `make lint`           | Lint all code                  |
| `make format`         | Format all code (Go + JS/TS)   |
| `make format-check`   | Check formatting (for CI)      |
| `make clean`          | Clean build artifacts          |

## Type Generation

Types flow from the OpenAPI spec (`api/openapi.yaml`):

1. **Go server** — `oapi-codegen` generates typed interfaces in `api/internal/openapi/server.gen.go`
2. **TypeScript** — `openapi-typescript` generates types in `packages/api-client/generated/schema.ts`

After changing the API spec, regenerate both:

```bash
make generate-types
```

## Testing

```bash
# Run all tests
make test

# Go tests only
cd api && go test ./...

# Go tests with verbose output
cd api && go test -v ./...

# Specific package
cd api && go test ./internal/user/...
```

## Code Style

```bash
# Lint all code
make lint

# Format all code
make format

# Check formatting (used in CI)
make format-check
```

## Tech Stack

| Component     | Technology                                        |
| ------------- | ------------------------------------------------- |
| Backend       | Go, Chi, SQLite                                   |
| Frontend      | React, TypeScript, Vite, TanStack Query, Tailwind |
| UI Components | React Aria Components, tailwind-variants          |
| Desktop       | Electron, Electron Forge                          |
| Real-time     | Server-Sent Events                                |
| Types         | OpenAPI 3.0, oapi-codegen, openapi-typescript     |

## Submitting Changes

Before writing code, please [open an issue](https://github.com/enzyme/enzyme/issues) to discuss your idea. This helps avoid duplicate effort and lets us align on the approach before you invest time in a PR.

Once an issue is agreed upon:

1. Fork the repository
2. Create a feature branch (`git checkout -b my-feature`)
3. Make your changes
4. Run `make lint` and `make test` to verify
5. Commit your changes
6. Push to your fork and open a pull request referencing the issue

Please keep pull requests focused on a single change to make review easier.
