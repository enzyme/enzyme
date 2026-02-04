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
│   ├── oapi-codegen.yaml       # Code generation config
│   ├── cmd/feather/main.go     # Entry point, CLI flags, graceful shutdown
│   ├── internal/
│   │   ├── api/                # Generated code (DO NOT EDIT)
│   │   │   ├── generate.go     # go:generate directive
│   │   │   └── server.gen.go   # Generated types, interfaces, handlers
│   │   ├── handler/            # Handler implementations
│   │   │   ├── handler.go      # Main Handler struct (implements StrictServerInterface)
│   │   │   ├── auth.go         # Auth endpoint implementations
│   │   │   ├── workspace.go    # Workspace endpoint implementations
│   │   │   ├── channel.go      # Channel endpoint implementations
│   │   │   ├── message.go      # Message endpoint implementations
│   │   │   ├── file.go         # File endpoint implementations
│   │   │   └── errors.go       # Shared error helpers
│   │   ├── app/app.go          # Dependency wiring, startup sequence
│   │   ├── config/             # Configuration loading and validation
│   │   ├── database/           # SQLite connection, migrations
│   │   ├── auth/               # Authentication (sessions, middleware)
│   │   ├── user/               # User domain (model, repository)
│   │   ├── workspace/          # Workspace domain (model, repository)
│   │   ├── channel/            # Channel domain (model, repository)
│   │   ├── message/            # Message domain (model, repository)
│   │   ├── file/               # File uploads (model, repository)
│   │   ├── sse/                # Server-Sent Events (hub, handlers - manual)
│   │   ├── presence/           # Online/away/offline tracking
│   │   ├── email/              # Email service + templates
│   │   └── server/             # HTTP server, router (chi)
│   └── Makefile
├── clients/
│   └── web/                    # React frontend (@feather/web)
│       └── src/
│           ├── api/            # API client functions
│           ├── hooks/          # React Query hooks + UI state hooks
│           ├── components/     # UI components
│           ├── pages/          # Route pages
│           └── lib/            # Utilities, SSE client, presence store
├── packages/
│   └── api-client/             # Shared types package (@feather/api-client)
│       ├── src/                # Type aliases, fetch client
│       └── generated/          # Auto-generated from OpenAPI
├── package.json                # pnpm workspace root
├── pnpm-workspace.yaml
└── Makefile
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
1. **Go server**: `oapi-codegen` generates `api/internal/openapi/server.gen.go` (types + strict server interface)
2. **TypeScript types**: `openapi-typescript` generates `packages/api-client/generated/schema.ts`

Regenerate after API changes:
```bash
make generate-types    # Regenerate both Go and TypeScript
cd api && make generate  # Regenerate Go only
```

The Go server uses spec-first development:
- `api/openapi.yaml` is the single source of truth
- `oapi-codegen` generates `StrictServerInterface` with typed request/response objects
- Handler implementations in `api/internal/handler/` must satisfy the interface (compiler enforces)
- SSE endpoints are excluded from generation (require streaming)

Usage in web client:
```typescript
import type { User, Message, Channel } from '@feather/api-client';
import { get, post, ApiError } from '@feather/api-client';
```

---

## API (Go Backend)

### Architecture Patterns

**Spec-First OpenAPI** - The Go server uses generated code from OpenAPI:
- `api/openapi.yaml` defines all endpoints, request/response schemas
- `oapi-codegen` generates `StrictServerInterface` with typed wrappers
- `api/internal/handler/` contains implementations (one file per domain)
- Compiler enforces the contract - drift is impossible

**Domain Structure** - Each domain (user, workspace, channel, message, file) follows:
- `model.go` - Data structures and constants
- `repository.go` - Database operations
- Handler implementations in `api/internal/handler/<domain>.go`

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
| `api/openapi.yaml` | API specification (source of truth) |
| `api/internal/openapi/server.gen.go` | Generated types and interfaces |
| `api/internal/handler/handler.go` | Main handler implementing StrictServerInterface |
| `api/internal/app/app.go` | Dependency wiring |
| `api/internal/server/router.go` | Router setup, mounts generated handlers |
| `api/internal/auth/middleware.go` | Auth middleware (used for SSE routes) |
| `api/internal/sse/hub.go` | Real-time broadcasting |

### Common API Tasks

**Add endpoint**:
1. Add endpoint to `api/openapi.yaml` with request/response schemas
2. Run `cd api && make generate` to regenerate `server.gen.go`
3. Implement the new method in the appropriate `api/internal/handler/<domain>.go`
4. The compiler will error if the implementation is missing or has wrong signature

**Add migration**: Create `api/internal/database/migrations/NNN_description.sql` with `-- +goose Up/Down`

**Add domain**: Create package in `internal/`, add model/repository/handler, wire in `app.go`

### Configuration

Loads in order (later overrides earlier):
1. Defaults (`config.Defaults()`)
2. Config file (`config.yaml` or `--config`)
3. Environment (`FEATHER_` prefix)
4. CLI flags

### Testing

**Run tests**:
```bash
cd api && go test ./...           # Run all tests
cd api && go test -v ./...        # Verbose output
cd api && go test -cover ./...    # With coverage
cd api && go test ./internal/user/...  # Specific package
```

**Test file conventions**:
- Test files are named `*_test.go` alongside source files
- Test database uses in-memory SQLite (`:memory:`)
- Use low bcrypt cost (4) in tests for speed

**Test utilities** (`api/internal/testutil/`):
```go
// In-memory test database with migrations
db := testutil.TestDB(t)

// Create test fixtures
user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
ws := testutil.CreateTestWorkspace(t, db, user.ID, "slug", "Name")
ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")
msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Hello")

// Mock password reset repository for auth service tests
mockResets := testutil.NewMockPasswordResetRepository()
```

**Test patterns**:
- Table-driven tests for permission helpers and validation
- Repository tests use real in-memory SQLite
- Handler tests use minimal dependencies with mock repositories

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
- **UI state**: URL search params (thread/profile panels via `usePanel.ts`) and localStorage (sidebar via `useSidebar.ts`)
- **Ephemeral state**: `useSyncExternalStore` in `lib/presenceStore.ts` for typing indicators and presence

**Real-time**: SSE in `src/lib/sse.ts`, `useSSE` hook updates React Query cache on events

**Styling**: Tailwind CSS, dark mode via `dark:` prefix

**UI Components**: React Aria Components (RAC) for accessible, keyboard-navigable UI primitives
- Wrapper components in `src/components/ui/` use RAC + `tailwind-variants` for styling
- RAC provides focus management, keyboard nav, ARIA attributes out of the box
- Use `onPress` instead of `onClick`, `isDisabled` instead of `disabled`
- Use `cn()` from `lib/utils` for conditional class merging (re-exported from `tailwind-variants`)

### Key Web Files

| File | Purpose |
|------|---------|
| `src/App.tsx` | Router, providers |
| `src/hooks/useAuth.ts` | Auth state |
| `src/hooks/useMessages.ts` | Messages, reactions |
| `src/hooks/useSSE.ts` | SSE → cache updates |
| `src/hooks/usePanel.ts` | Thread/profile panel state (URL-based) |
| `src/lib/presenceStore.ts` | Typing indicators, user presence |

### Common Web Tasks

**Add API endpoint**:
1. Add to `api/openapi.yaml`
2. Run `make generate-types`
3. Add function in `src/api/`
4. Create hook with `useQuery`/`useMutation`

**Add SSE event**: Add to OpenAPI SSEEventType enum, regenerate, add handler in `useSSE.ts`

**Add page**: Create in `src/pages/`, add route in `App.tsx`, wrap with `<RequireAuth>` if needed

**Add UI component**: Create in `src/components/ui/`, use React Aria Components + `tv()` for styling:
```tsx
import { Button as AriaButton } from 'react-aria-components';
import { tv } from 'tailwind-variants';

const styles = tv({ base: '...', variants: { size: { sm: '...', md: '...' } } });
export function MyComponent({ size = 'md' }) {
  return <AriaButton className={styles({ size })}>...</AriaButton>;
}
```
Export from `src/components/ui/index.ts`

### Patterns

**Optimistic updates** (see `useAddReaction`):
1. `onMutate`: Cancel queries, save previous, update cache
2. `onError`: Rollback
3. `onSettled`: Invalidate

**Infinite scroll**: `MessageList` uses `useInfiniteQuery`, messages reversed for display, scroll preserved

**Thread panel**: Slide-out controlled by `?thread=` URL search param (see `useThreadPanel` hook)