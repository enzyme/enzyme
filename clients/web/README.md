# Feather Web Client

React frontend for Feather, a self-hostable Slack alternative.

> Part of the [Feather monorepo](../../README.md). See root for full setup.

## Tech Stack

- **React 18+** with hooks
- **TypeScript** for type safety
- **Vite** for fast development and bundling
- **TanStack Query** for server state (caching, fetching, background refetch)
- **Zustand** for client state (UI state, presence)
- **Tailwind CSS** for styling
- **React Router v6** for routing

## Getting Started

### Prerequisites

- Node.js 18+
- Backend server running at `http://localhost:8080`

### Development

From monorepo root:
```bash
make dev  # Starts API and web together
```

Or just the web client:
```bash
pnpm --filter @feather/web dev
```

The app runs at `http://localhost:3000` and proxies `/api` requests to the backend.

### Build

```bash
pnpm --filter @feather/web build
```

## Project Structure

```
src/
├── api/              # API endpoint modules
│   ├── auth.ts       # Login, register, logout, me
│   ├── workspaces.ts # Workspace CRUD, members, invites
│   ├── channels.ts   # Channel CRUD, members, join/leave
│   └── messages.ts   # Messages CRUD, reactions, threads
├── hooks/            # Custom React hooks
│   ├── useAuth.ts    # Auth state + login/register/logout mutations
│   ├── useSSE.ts     # SSE connection with React Query cache updates
│   ├── useMessages.ts # Infinite scroll + optimistic updates
│   ├── useTyping.ts  # Debounced typing indicators
│   └── useAutoScroll.ts # Scroll position management
├── stores/           # Zustand stores
│   ├── uiStore.ts    # Sidebar state, active thread, dark mode
│   └── presenceStore.ts # User presence + typing indicators
├── components/
│   ├── ui/           # Reusable UI components
│   ├── auth/         # LoginForm, RegisterForm, RequireAuth
│   ├── workspace/    # WorkspaceSwitcher with create modal
│   ├── channel/      # ChannelSidebar with create modal
│   ├── message/      # MessageList, MessageItem, MessageComposer, ReactionPicker
│   ├── thread/       # ThreadPanel (slide-out side panel)
│   └── layout/       # AppLayout (3-column layout)
├── pages/            # Route page components
└── lib/              # Utilities and shared code
    ├── queryClient.ts # TanStack Query client config
    ├── sse.ts        # SSE connection class with auto-reconnect
    └── utils.ts      # Helper functions
```

Types are imported from the shared `@feather/api-client` package (see `../../packages/api-client`).

## Features

### Authentication
- Login and registration forms
- Session-based auth with cookies
- Protected routes with `RequireAuth` guard
- Auto-redirect based on auth state

### Workspaces
- Workspace switcher in left sidebar
- Create new workspaces
- Invite link generation
- Accept invites via `/invites/:code`

### Channels
- Channel sidebar grouped by type (public, private, DMs)
- Create public/private channels
- Join/leave channels
- Unread count badges

### Messaging
- Infinite scroll with cursor-based pagination
- Messages loaded newest-first, displayed oldest-at-top
- Auto-scroll to bottom on new messages (when already at bottom)
- Scroll position preserved when loading older messages
- Message editing and deletion

### Real-time Updates
- SSE connection to `/api/workspaces/{id}/events`
- Auto-reconnect with 3-second delay on disconnect
- Events update React Query cache directly (no refetch flicker)
- Connection status indicator

### Reactions
- Emoji reaction picker
- Optimistic updates for instant feedback
- Add/remove reactions
- Grouped reaction display with counts

### Threads
- Slide-out side panel (main channel stays visible)
- Thread replies with infinite scroll
- Reply count on parent messages
- Separate composer for thread replies

### Typing Indicators
- Debounced typing start (1 second)
- Auto-stop after 3 seconds of inactivity
- Stop on blur or message submit
- Display in message composer area

### UI/UX
- Dark mode with system preference detection
- Collapsible sidebar (responsive)
- Loading skeletons
- Toast notifications
- Custom scrollbars

## Routes

| Route | Component | Description |
|-------|-----------|-------------|
| `/login` | LoginPage | Login form |
| `/register` | RegisterPage | Registration form |
| `/invites/:code` | AcceptInvitePage | Accept workspace invite |
| `/workspaces` | WorkspaceListPage | Workspace selection |
| `/workspaces/:workspaceId` | AppLayout | Redirects to first channel |
| `/workspaces/:workspaceId/channels/:channelId` | ChannelPage | Main chat view |

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Server state | TanStack Query | Automatic caching, background refetch, request deduplication |
| Client state | Zustand | Simple, minimal boilerplate, no context providers needed |
| Styling | Tailwind CSS | Utility-first, no component library overhead, full control |
| Real-time | SSE → update cache | Instant UI updates without refetch flicker |
| Threads | Side panel | Matches Slack UX, main channel stays visible |
| Auth | Session cookies | Backend handles session; `credentials: 'include'` on all requests |

## API Integration

API calls use the shared client from `@feather/api-client`:
```typescript
import { get, post, ApiError } from '@feather/api-client';
import type { User, Message } from '@feather/api-client';
```

- `credentials: 'include'` for session cookies
- Automatic error handling with `ApiError` class
- Type-safe responses generated from OpenAPI spec

### SSE Events Handled

| Event | Action |
|-------|--------|
| `message.new` | Prepend to messages cache |
| `message.updated` | Update message in cache |
| `message.deleted` | Remove from cache |
| `reaction.added` | Add reaction to message |
| `reaction.removed` | Remove reaction from message |
| `channel.*` | Invalidate channel list |
| `typing.start/stop` | Update presence store |
| `presence.changed` | Update user presence |

## Environment

The Vite dev server proxies `/api` to `http://localhost:8080`. For production, configure your server to proxy or serve the API from the same origin.
