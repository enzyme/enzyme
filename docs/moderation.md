# Moderation

Enzyme provides workspace-level moderation tools for managing disruptive behavior and giving users control over their experience. This guide covers banning, blocking, message pinning, and the audit log.

See [Permissions & Roles](permissions.md#moderation) for the permission matrix.

## Banning

Workspace owners and admins can ban users from a workspace. Banning immediately removes the user's membership, disconnects their active sessions, and prevents them from rejoining via invite links.

### What happens when a user is banned

1. Their workspace membership is removed.
2. Their channel memberships are removed (except DM channels — the other participant retains access to conversation history).
3. Their active SSE connections are terminated.
4. They cannot accept new invite links for that workspace.

Unbanning removes the ban record but does **not** restore the user's membership. The user must rejoin via a new invite link.

### Role hierarchy

Bans follow a strict role hierarchy. You can only ban users with a lower role rank than yours:

| Actor     | Can ban              |
| --------- | -------------------- |
| **Owner** | Admin, Member, Guest |
| **Admin** | Member, Guest        |

The owner can never be banned. Self-banning is rejected.

### Temporary bans

Bans can include an expiry duration (in hours). Expired bans are checked lazily — the ban is considered inactive once `expires_at` passes, and the user can accept new invites. No background job is needed.

Available preset durations: 1 hour, 24 hours, 7 days, 30 days, or permanent.

### Hiding messages

When banning a user, admins can optionally choose to **hide the user's messages** from other members. When enabled:

- The banned user's messages are filtered from all message queries (channel messages, threads, search, unread counts, pinned messages).
- Their reactions and thread participations are also hidden.
- Messages are **not deleted** — they are filtered at query time. Unbanning (or un-hiding) immediately restores visibility.

### Ban enforcement on write endpoints

Banned users are blocked from sending messages, editing messages, and adding reactions. These endpoints return a `403 Forbidden` response if the user has an active ban in the workspace.

## Blocking

Any workspace member can block another member within a workspace. Blocks are **invisible** to the blocked user — they receive no notification and can still send messages (which the blocker will not see).

### Role restrictions

Users cannot block members with the `admin` or `owner` role. This prevents regular members from silencing workspace leadership. Self-blocking is rejected.

### What blocking does

| Behavior                      | Details                                                                                                                                                             |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Messages hidden**           | The blocked user's messages are filtered from all channels for the blocker (channel messages, threads, search, unread counts, pinned messages).                     |
| **Reactions hidden**          | The blocked user's reactions and thread participations are hidden from the blocker.                                                                                 |
| **DM messages not delivered** | When the blocked user sends a DM, the message is stored normally but the SSE event is not delivered to the blocker. The blocker's message list also filters it out. |
| **DM creation prevented**     | Neither the blocker nor the blocked user can create a new DM with the other. Adding a blocked user to a group DM is also prevented.                                 |
| **Mentions stripped**         | @mentions are stripped in both directions — the blocker won't receive mention notifications from the blocked user, and vice versa.                                  |

### What blocking does not do

- The blocked user **can still see the blocker's messages**. Blocking is one-directional.
- Existing DM channels are preserved. The blocked user can continue sending messages into the DM (the blocker just won't see them).
- Blocking in one workspace does **not** affect other workspaces. Blocks are workspace-scoped.

### Unblocking

Unblocking immediately restores visibility of all historical messages from the previously blocked user. There is no role restriction on unblocking — you can always undo your own blocks.

## Message Pinning

Channel members who can post can pin and unpin messages. Workspace admins can also pin messages in public channels without explicit channel membership.

### Limits

Each channel has a maximum of **50 pinned messages**. This limit is enforced atomically to prevent race conditions.

### Behavior

- Pinning or unpinning a message creates a system message in the channel (e.g., "Alice pinned a message").
- Deleted messages cannot be pinned.
- Pinned messages from banned users with hidden messages are filtered from the pinned list.
- The pinned messages panel shows all pins in a channel, ordered by pin date (newest first).

## Audit Log

All moderation actions are recorded in a per-workspace audit log. Only owners and admins can view it.

### Logged actions

| Action                | Trigger                                            |
| --------------------- | -------------------------------------------------- |
| `user.banned`         | A user is banned from the workspace                |
| `user.unbanned`       | A ban is removed                                   |
| `member.removed`      | An admin removes another member (not self-removal) |
| `member.role_changed` | A member's role is changed                         |
| `message.deleted`     | An admin deletes another user's message (not own)  |
| `channel.archived`    | A channel is archived                              |

Each entry records the actor, action, target, timestamp, and optional metadata (e.g., ban reason, duration, old/new role, original message content for admin deletes).
