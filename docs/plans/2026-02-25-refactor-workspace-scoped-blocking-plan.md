---
title: "refactor: Make user blocking workspace-scoped with role gating"
type: refactor
status: completed
date: 2026-02-25
origin: docs/plans/2026-02-25-feat-moderation-tools-plan.md
---

# refactor: Make User Blocking Workspace-Scoped with Role Gating

## Overview

Refactor user blocking from cross-workspace (global per-user) to workspace-scoped, and add role gating so that users cannot block workspace admins or owners. This prevents users from pre-emptively blocking moderators to avoid oversight.

The original brainstorm chose cross-workspace blocking ("a block should follow you"), but in practice most users will be in one or two workspaces, and workspace-scoped blocking aligns with how bans already work. More importantly, workspace-scoping enables role gating — the block endpoint now has workspace context to check roles.

## Problem Statement / Motivation

Under cross-workspace blocking, any user can block any other user — including workspace admins and owners. A bad actor could block all admins before engaging in disruptive behavior, preventing admins from seeing their activity (once block enforcement is wired into message queries). Workspace-scoped blocking with role gating closes this loophole.

## Proposed Solution

1. Recreate `user_blocks` table with a `workspace_id` column
2. Move block endpoints from `/users/blocks/...` to `/workspaces/{wid}/blocks/...`
3. Add membership verification and role gating in the handler
4. Update all repository methods, models, tests, and frontend code

## Technical Approach

### Database Migration

New migration `039_workspace_scope_user_blocks.sql`:

```sql
-- +goose Up
DROP TABLE IF EXISTS user_blocks;

CREATE TABLE user_blocks (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (workspace_id, blocker_id, blocked_id)
);

CREATE INDEX idx_user_blocks_workspace_blocker ON user_blocks(workspace_id, blocker_id);
CREATE INDEX idx_user_blocks_workspace_blocked ON user_blocks(workspace_id, blocked_id);

-- +goose Down
DROP TABLE IF EXISTS user_blocks;

CREATE TABLE user_blocks (
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (blocker_id, blocked_id)
);

CREATE INDEX idx_user_blocks_blocker ON user_blocks(blocker_id);
CREATE INDEX idx_user_blocks_blocked ON user_blocks(blocked_id);
```

No data migration needed — this is v0.1.0 pre-release with no production instances.

### Role Gating Rules

- **Cannot block**: users with `admin` or `owner` role in the workspace
- **Can block**: `member` and `guest` roles (any role can initiate a block on these targets)
- **Cannot block yourself** (existing rule, preserved)
- **Unblocking**: no role restriction — you can always undo your own block
- **Role promotion after block**: existing blocks persist. The role check only applies at block-creation time. This is consistent with how bans work (banning checks role hierarchy, but a previously banned user becoming admin elsewhere doesn't auto-unban them)
- **Membership required**: both blocker and target must be members of the workspace

### Handler Changes

`api/internal/handler/moderation.go` — `BlockUser`:

1. Extract `workspaceID` from path parameter (`request.Wid`)
2. Verify blocker is a workspace member (`h.workspaceRepo.GetMembership`)
3. Verify target is a workspace member
4. Check target role: reject if `admin` or `owner` (return 403)
5. Call `h.moderationRepo.CreateBlock(ctx, workspaceID, userID, targetUserID)`

`UnblockUser`: Add workspace_id, verify membership, no role check.

`ListBlocks`: Add workspace_id, verify membership, filter by workspace.

### OpenAPI Spec Changes

Remove:
- `POST /users/blocks/create`
- `POST /users/blocks/remove`
- `POST /users/blocks/list`

Add (under existing workspace group):
- `POST /workspaces/{wid}/blocks/create` — body: `{ user_id: string }`
- `POST /workspaces/{wid}/blocks/remove` — body: `{ user_id: string }`
- `GET /workspaces/{wid}/blocks` — returns `{ blocks: BlockWithUser[] }`

Update `BlockWithUser` schema to include `workspace_id`.

### Repository Changes

All six block methods gain a `workspaceID` parameter:

| Method | New Signature |
|--------|--------------|
| `CreateBlock` | `(ctx, workspaceID, blockerID, blockedID) error` |
| `DeleteBlock` | `(ctx, workspaceID, blockerID, blockedID) error` |
| `ListBlocks` | `(ctx, workspaceID, blockerID) ([]BlockWithUser, error)` |
| `GetBlockedUserIDs` | `(ctx, workspaceID, blockerID) (map[string]bool, error)` |
| `IsBlocked` | `(ctx, workspaceID, blockerID, blockedID) (bool, error)` |
| `IsBlockedEitherDirection` | `(ctx, workspaceID, userA, userB) (bool, error)` |

All SQL queries add `AND workspace_id = ?`.

### Model Changes

`api/internal/moderation/model.go`:

```go
type Block struct {
    WorkspaceID string    `json:"workspace_id"`
    BlockerID   string    `json:"blocker_id"`
    BlockedID   string    `json:"blocked_id"`
    CreatedAt   time.Time `json:"created_at"`
}

type BlockWithUser struct {
    Block
    DisplayName string  `json:"display_name"`
    Email       string  `json:"email"`
    AvatarURL   *string `json:"avatar_url,omitempty"`
}
```

### Frontend Changes

**API client** (`apps/web/src/api/moderation.ts`):
```typescript
blockUser: (workspaceId: string, userId: string) =>
  post(`/workspaces/${workspaceId}/blocks/create`, { user_id: userId }),
unblockUser: (workspaceId: string, userId: string) =>
  post(`/workspaces/${workspaceId}/blocks/remove`, { user_id: userId }),
listBlocks: (workspaceId: string) =>
  get(`/workspaces/${workspaceId}/blocks`),
```

**Hooks** (`apps/web/src/hooks/useModeration.ts`):
- `useBlocks(workspaceId)` — query key `['user-blocks', workspaceId]`
- `useBlockUser(workspaceId)` — invalidates `['user-blocks', workspaceId]`
- `useUnblockUser(workspaceId)` — invalidates `['user-blocks', workspaceId]`

**ProfilePane** (`apps/web/src/components/profile/ProfilePane.tsx`):
- Pass `workspaceId` from route params to block hooks
- Block button shows "Block in this workspace" or simply "Block User" (operates on current workspace)

**MessageItem** (`apps/web/src/components/message/MessageItem.tsx`):
- `useBlockUser` call needs workspace context (available from parent via props or route)

### Files to Modify

| File | Change |
|------|--------|
| `api/internal/database/migrations/039_*.sql` | New migration (create) |
| `api/openapi.yaml` | Move block endpoints, update schema |
| `api/internal/openapi/server.gen.go` | Regenerated |
| `api/internal/moderation/model.go` | Add `WorkspaceID` to Block |
| `api/internal/moderation/repository.go` | Add workspace_id to all 6 methods |
| `api/internal/moderation/repository_test.go` | Rewrite all block tests with workspace context |
| `api/internal/handler/moderation.go` | Rework BlockUser/UnblockUser/ListBlocks handlers |
| `packages/api-client/generated/schema.ts` | Regenerated |
| `apps/web/src/api/moderation.ts` | Update block API functions |
| `apps/web/src/hooks/useModeration.ts` | Add workspaceId to block hooks |
| `apps/web/src/components/profile/ProfilePane.tsx` | Pass workspace context |
| `apps/web/src/components/message/MessageItem.tsx` | Pass workspace context to block |
| `docs/permissions.md` | Update blocking section |

## Acceptance Criteria

### Backend
- [x] Migration `039` recreates `user_blocks` with `workspace_id` in composite PK
- [x] `CreateBlock` requires workspace membership for both users
- [x] `CreateBlock` rejects blocking admin/owner targets (403)
- [x] `CreateBlock` still prevents self-blocking (400)
- [x] `CreateBlock` remains idempotent
- [x] `DeleteBlock` works with workspace_id, no role restriction
- [x] `ListBlocks` returns blocks scoped to the given workspace
- [x] `GetBlockedUserIDs` filters by workspace_id
- [x] `IsBlocked` and `IsBlockedEitherDirection` filter by workspace_id
- [x] All existing Go tests pass

### Frontend
- [x] Block/unblock API calls include workspace_id in URL path
- [x] Block hooks accept and use workspaceId parameter
- [x] ProfilePane block button operates on current workspace
- [x] MessageItem block action operates on current workspace
- [x] Query cache keys include workspaceId

### OpenAPI & Types
- [x] Old `/users/blocks/*` endpoints removed from spec
- [x] New `/workspaces/{wid}/blocks/*` endpoints added
- [x] `BlockWithUser` schema includes `workspace_id`
- [x] TypeScript types regenerated

### Tests
- [x] All block repository tests rewritten with workspace context
- [x] Role gating test: member cannot block admin
- [x] Role gating test: member cannot block owner
- [x] Role gating test: admin can block member
- [x] Membership test: non-member cannot block in workspace
- [x] Unblock test: no role restriction on unblocking

### Documentation
- [x] `docs/permissions.md` blocking section updated with role restrictions

## Sources & References

### Origin
- **Parent plan:** [docs/plans/2026-02-25-feat-moderation-tools-plan.md](docs/plans/2026-02-25-feat-moderation-tools-plan.md) — original moderation tools plan chose cross-workspace blocking; this refactoring reverses that decision
- **Brainstorm:** [docs/brainstorms/2026-02-25-moderation-tools-brainstorm.md](docs/brainstorms/2026-02-25-moderation-tools-brainstorm.md) — "Block persistence: Per-user (cross-workspace)" decision being revised

### Internal References
- Ban handler pattern (role gating model): `api/internal/handler/moderation.go:16-146`
- Workspace role helpers: `api/internal/workspace/model.go:83-124`
- Current block migration: `api/internal/database/migrations/037_create_user_blocks.sql`
- Current block repository: `api/internal/moderation/repository.go:192-294`
- Current block tests: `api/internal/moderation/repository_test.go:305-500`
