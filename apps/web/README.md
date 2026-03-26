# Enzyme Web Client

React frontend for Enzyme, a self-hostable Slack alternative.

> Part of the [Enzyme monorepo](../../README.md). See root for full setup.

## Tech Stack

- **React 18+** with hooks
- **TypeScript** for type safety
- **Vite** for fast development and bundling
- **TanStack Query** for server state (caching, fetching, background refetch)
- **React Aria Components** for accessible UI primitives (buttons, inputs, modals, menus, tabs)
- **tailwind-variants** for component styling with variants
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
pnpm --filter @enzyme/web dev
```

The app runs at `http://localhost:3000` and proxies `/api` requests to the backend.

### Build

```bash
pnpm --filter @enzyme/web build
```

## Project Structure

```
src/
├── hooks/            # Custom React hooks
│   ├── useAuth.ts    # Auth state + login/register/logout mutations
│   ├── useSSE.ts     # SSE connection with React Query cache updates
│   ├── useMessages.ts # Infinite scroll + optimistic updates
│   ├── useTyping.ts  # Debounced typing indicators
│   ├── usePanel.ts   # URL-based thread/profile panel state
│   ├── useSidebar.ts # localStorage-based sidebar state
│   └── useAutoScroll.ts # Scroll position management
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
    ├── presenceStore.ts # Typing indicators + presence (useSyncExternalStore)
    └── utils.ts      # Helper functions
```

Types are imported from the shared `@enzyme/api-client` package (see `../../packages/api-client`).

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

### UI Components (React Aria)

All interactive components use React Aria Components for built-in accessibility:

| Component    | Description                                                         |
| ------------ | ------------------------------------------------------------------- |
| `Button`     | Accessible button with variants (primary, secondary, ghost, danger) |
| `Input`      | Text field with label and error handling                            |
| `Modal`      | Dialog with focus trapping and backdrop                             |
| `Menu`       | Dropdown menu with keyboard navigation                              |
| `Tabs`       | Tab list with panels                                                |
| `RadioGroup` | Radio button group                                                  |

Usage:

```tsx
import { Button, Input, Modal, Menu, MenuItem } from '../components/ui';

// Button uses onPress, not onClick
<Button variant="primary" onPress={() => save()} isLoading={saving}>
  Save
</Button>

// Input uses isRequired, not required
<Input label="Email" value={email} onChange={e => setEmail(e.target.value)} isRequired />
```

## Routes

| Route                                          | Component          | Description                                     |
| ---------------------------------------------- | ------------------ | ----------------------------------------------- |
| `/login`                                       | LoginPage          | Login form                                      |
| `/register`                                    | RegisterPage       | Registration form (includes workspace creation) |
| `/invites/:code`                               | AcceptInvitePage   | Accept workspace invite                         |
| `/workspaces/:workspaceId`                     | AppLayout          | Redirects to first channel                      |
| `/workspaces/:workspaceId/channels/:channelId` | ChannelPage        | Main chat view                                  |
| `/settings`                                    | ServerSettingsPage | Server settings                                 |

## Architecture Decisions

| Decision        | Choice                           | Rationale                                                                    |
| --------------- | -------------------------------- | ---------------------------------------------------------------------------- |
| Server state    | TanStack Query                   | Automatic caching, background refetch, request deduplication                 |
| UI state        | URL search params + localStorage | Shareable URLs for panels, persisted preferences, zero dependencies          |
| Ephemeral state | `useSyncExternalStore`           | Built-in React 18 API for presence/typing, no external libraries             |
| UI primitives   | React Aria Components            | Accessible by default, keyboard nav, focus management, ARIA built-in         |
| Styling         | Tailwind + tailwind-variants     | Utility-first with variant support via `tv()`, class merging via `cn()`      |
| Real-time       | SSE → update cache               | Instant UI updates without refetch flicker                                   |
| Threads         | Side panel                       | Matches Slack UX, main channel stays visible                                 |
| Auth            | Bearer tokens                    | Token stored in localStorage; `Authorization: Bearer` header on all requests |

## API Integration

API calls use `openapi-fetch` via the shared client from `@enzyme/api-client`:

```typescript
import { apiClient, throwIfError, ApiError } from '@enzyme/api-client';
import type { User, Message } from '@enzyme/api-client';

// Type-safe API calls — paths, methods, and payloads checked at compile time
const channel = await throwIfError(
  apiClient.POST('/api/channels/{id}/update', {
    params: { path: { id } },
    body: { name: 'new-name' },
  }),
);
```

- `Authorization: Bearer` header attached automatically via middleware
- `throwIfError` unwraps responses and throws `ApiError` on failure
- Paths and request/response types are generated from the OpenAPI spec

### SSE Events Handled

| Event                     | Action                                        |
| ------------------------- | --------------------------------------------- |
| `message.new`             | Prepend to messages cache                     |
| `message.updated`         | Update message in cache                       |
| `message.deleted`         | Remove from cache                             |
| `message.pinned/unpinned` | Update pin state in cache                     |
| `reaction.added`          | Add reaction to message                       |
| `reaction.removed`        | Remove reaction from message                  |
| `channel.created`         | Optimistically insert into channel list cache |
| `channel.updated`         | Invalidate channel list                       |
| `channel.archived`        | Optimistically remove from channel list cache |
| `channels.invalidate`     | Invalidate channel list (batch operations)    |
| `channel.member_*`        | Invalidate channel members                    |
| `typing.start/stop`       | Update typing state in `lib/presenceStore`    |
| `presence.changed`        | Update user presence in `lib/presenceStore`   |
| `notification`            | Show desktop notification                     |
| `emoji.*`                 | Invalidate custom emoji cache                 |
| `member.*`                | Invalidate workspace members                  |
| `workspace.updated`       | Invalidate workspace data                     |
| `scheduled_message.*`     | Invalidate scheduled messages cache           |

## Environment

The Vite dev server proxies `/api` to `http://localhost:8080`. For production, configure your server to proxy or serve the API from the same origin.
