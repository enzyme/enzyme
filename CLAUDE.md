# Feather Monorepo

Self-hostable Slack alternative with Go backend and React frontend.

## Quick Start

```bash
# Install dependencies
make install

# Start development servers (API + Web concurrently)
make dev

# Or run separately
cd api && make dev              # Go server on :8080
pnpm --filter @feather/web dev  # React on :3000
```

## Project Structure

```
feather/
├── api/                        # Go backend server
│   ├── openapi.yaml            # API specification (source of truth)
│   ├── cmd/feather/main.go     # Entry point, CLI flags, graceful shutdown
│   ├── internal/
│   │   ├── app/app.go          # Dependency wiring, startup sequence
│   │   ├── config/             # Configuration loading and validation
│   │   ├── database/           # SQLite connection, migrations
│   │   ├── auth/               # Authentication (handlers, sessions, middleware)
│   │   ├── user/               # User domain
│   │   ├── workspace/          # Workspace domain
│   │   ├── channel/            # Channel domain
│   │   ├── message/            # Message domain
│   │   ├── file/               # File uploads
│   │   ├── sse/                # Server-Sent Events (hub, handlers)
│   │   ├── presence/           # Online/away/offline tracking
│   │   ├── email/              # Email service + templates
│   │   └── server/             # HTTP server, router (chi)
│   └── Makefile
├── clients/
│   └── web/                    # React frontend (@feather/web)
│       └── src/
│           ├── api/            # API client functions
│           ├── hooks/          # React Query hooks
│           ├── stores/         # Zustand stores (uiStore, presenceStore)
│           ├── components/     # UI components
│           ├── pages/          # Route pages
│           └── lib/            # Utilities, SSE client
├── packages/
│   └── api-client/             # Shared types package (@feather/api-client)
│       ├── src/                # Type aliases, fetch client
│       └── generated/          # Auto-generated from OpenAPI
├── package.json                # pnpm workspace root
├── pnpm-workspace.yaml
├── Makefile
└── docker-compose.yml
```

## Build Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start API and web dev servers |
| `make build` | Build all (generate types first) |
| `make test` | Run all tests |
| `make generate-types` | Regenerate types from OpenAPI |
| `make lint` | Lint all code |
| `make clean` | Clean build artifacts |
| `make install` | Install all dependencies |

## Type Generation

Types flow from `api/openapi.yaml`:
1. **Go types**: `oapi-codegen` generates `api/internal/api/types.gen.go`
2. **TypeScript types**: `openapi-typescript` generates `packages/api-client/generated/schema.ts`

Regenerate after API changes:
```bash
make generate-types
```

Usage in web client:
```typescript
import type { User, Message, Channel } from '@feather/api-client';
import { get, post, ApiError } from '@feather/api-client';
```

---

## API (Go Backend)

### Architecture Patterns

**Domain Structure** - Each domain (user, workspace, channel, message, file) follows:
- `model.go` - Data structures and constants
- `repository.go` - Database operations
- `handler.go` - HTTP handlers

**Dependency Injection** - Wired in `api/internal/app/app.go`. The App struct owns all components.

**Database**
- SQLite with `modernc.org/sqlite` (pure Go, no CGO)
- Single connection (`SetMaxOpenConns(1)`) to avoid SQLITE_BUSY
- WAL mode, migrations via goose, timestamps as RFC3339

**Authentication**
- Session-based using `alexedwards/scs` with SQLite store
- bcrypt (cost 12), cookie named `feather_session`
- Get user: `auth.GetUserID(ctx)`

**IDs** - ULIDs via `ulid.Make().String()`

**Error Format**
```json
{"error": {"code": "ERROR_CODE", "message": "Human readable message"}}
```

**SSE** - Hub per workspace, 30s heartbeat, events stored for reconnection catch-up

### Key API Files

| File | Purpose |
|------|---------|
| `api/internal/app/app.go` | Dependency wiring |
| `api/internal/server/router.go` | All API routes |
| `api/internal/auth/middleware.go` | Auth middleware |
| `api/internal/sse/hub.go` | Real-time broadcasting |

### Common API Tasks

**Add endpoint**: Add handler in `handler.go`, register in `server/router.go`

**Add migration**: Create `api/internal/database/migrations/NNN_description.sql` with `-- +goose Up/Down`

**Add domain**: Create package in `internal/`, add model/repository/handler, wire in `app.go`

### Configuration

Loads in order (later overrides earlier):
1. Defaults (`config.Defaults()`)
2. Config file (`config.yaml` or `--config`)
3. Environment (`FEATHER_` prefix)
4. CLI flags

### Manual Testing

```bash
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","display_name":"Test"}'

# Login (save cookie)
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" -c cookies.txt \
  -d '{"email":"test@example.com","password":"password123"}'

# Authenticated request
curl -X POST http://localhost:8080/api/workspaces/create \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"slug":"my-workspace","name":"My Workspace"}'
```

---

## Web Client (React)

### Architecture

**State Management**
- **Server state**: TanStack Query - hooks in `src/hooks/`, cache keys like `['messages', channelId]`
- **Client state**: Zustand - `uiStore` (sidebar, threads, dark mode), `presenceStore` (typing, presence)

**Real-time**: SSE in `src/lib/sse.ts`, `useSSE` hook updates React Query cache on events

**Styling**: Tailwind CSS, dark mode via `dark:` prefix

### Key Web Files

| File | Purpose |
|------|---------|
| `src/App.tsx` | Router, providers |
| `src/hooks/useAuth.ts` | Auth state |
| `src/hooks/useMessages.ts` | Messages, reactions |
| `src/hooks/useSSE.ts` | SSE → cache updates |
| `src/stores/uiStore.ts` | UI state |

### Common Web Tasks

**Add API endpoint**:
1. Add to `api/openapi.yaml`
2. Run `make generate-types`
3. Add function in `src/api/`
4. Create hook with `useQuery`/`useMutation`

**Add SSE event**: Add to OpenAPI SSEEventType enum, regenerate, add handler in `useSSE.ts`

**Add page**: Create in `src/pages/`, add route in `App.tsx`, wrap with `<RequireAuth>` if needed

### Patterns

**Optimistic updates** (see `useAddReaction`):
1. `onMutate`: Cancel queries, save previous, update cache
2. `onError`: Rollback
3. `onSettled`: Invalidate

**Infinite scroll**: `MessageList` uses `useInfiniteQuery`, messages reversed for display, scroll preserved

**Thread panel**: Slide-out controlled by `activeThreadId` in `uiStore`

---

## Docker

```bash
docker-compose up --build
# api: :8080, web: :3000
```
