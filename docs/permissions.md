# Permissions & Roles

Enzyme uses role-based access control at two levels: **workspace** and **channel**. There are no server-level admin roles — all permissions are scoped to individual workspaces.

## Workspace Roles

Every workspace member has exactly one role. Roles are ordered by privilege:

| Role       | Rank | Description                               |
| ---------- | ---- | ----------------------------------------- |
| **Owner**  | 4    | Workspace creator, highest privilege      |
| **Admin**  | 3    | Can manage members and workspace settings |
| **Member** | 2    | Standard participant                      |
| **Guest**  | 1    | Limited access, cannot create channels    |

See [Workspace Administration](administration.md) for the operational guide on managing members, invites, and settings.

### Permission Matrix

| Action                              | Owner | Admin | Member | Guest |
| ----------------------------------- | :---: | :---: | :----: | :---: |
| Post messages (in allowed channels) |   ✓   |   ✓   |   ✓    |   ✓   |
| React to messages                   |   ✓   |   ✓   |   ✓    |   ✓   |
| Edit own messages                   |   ✓   |   ✓   |   ✓    |   ✓   |
| Delete own messages                 |   ✓   |   ✓   |   ✓    |   ✓   |
| Upload custom emoji                 |   ✓   |   ✓   |   ✓    |   ✓   |
| Delete own custom emoji             |   ✓   |   ✓   |   ✓    |   ✓   |
| Create channels                     |   ✓   |   ✓   |   ✓    |       |
| Convert group DM to channel         |   ✓   |   ✓   |   ✓    |       |
| Delete any message                  |   ✓   |   ✓   |        |       |
| Delete any custom emoji             |   ✓   |   ✓   |        |       |
| Delete any file                     |   ✓   |   ✓   |        |       |
| Manage members (add/remove)         |   ✓   |   ✓   |        |       |
| Change member roles                 |   ✓   |   ✓   |        |       |
| Create invite links                 |   ✓   |   ✓   |        |       |
| Update workspace name/settings      |   ✓   |   ✓   |        |       |
| Upload/remove workspace icon        |   ✓   |   ✓   |        |       |
| Archive channels                    |   ✓   |   ✓   |        |       |
| Promote member to admin             |   ✓   |       |        |       |
| Delete workspace                    |   ✓   |       |        |       |

### Special Rules

- **Owner is immutable**: The owner's role cannot be changed or transferred. The workspace creator is permanently the owner.
- **Admins cannot promote to admin**: Only the owner can make someone an admin. Admins can only assign the "member" role.
- **Self-removal**: Any member can remove themselves from a workspace, regardless of role. The owner cannot be removed.
- **Self role-change**: Users cannot change their own role.

### Permission Functions (Go)

These are defined in `api/internal/workspace/model.go`:

| Function               | Returns true for     | Used for                                                      |
| ---------------------- | -------------------- | ------------------------------------------------------------- |
| `CanManageMembers()`   | Owner, Admin         | Add/remove members, invites, settings, icon, archive channels |
| `CanChangeRole()`      | Owner, Admin         | Change member roles (with additional restrictions)            |
| `CanCreateChannels()`  | Owner, Admin, Member | Create channels, convert group DMs                            |
| `CanDeleteWorkspace()` | Owner                | Delete workspace (not yet implemented in handler)             |

## Channel Roles

Channel members can optionally have a channel-specific role. The role is nullable — `NULL` means default permissions (can post).

| Role       | Can Post | Can Manage Channel | Description                                  |
| ---------- | :------: | :----------------: | -------------------------------------------- |
| **admin**  |    ✓     |         ✓          | Can edit channel name/description/visibility |
| **poster** |    ✓     |                    | Can post messages                            |
| **viewer** |          |                    | Read-only access                             |
| _(null)_   |    ✓     |                    | Default when no role is set                  |

### Channel Permission Details

- **Update channel** (name, description, visibility): Requires channel admin role OR workspace owner/admin.
- **Add members**: Requires workspace owner/admin OR existing channel membership.
- **Archive channel**: Requires workspace owner/admin (channel admins cannot archive).
- **Public channels**: Non-members who are workspace members can post — they are auto-added with default (null) role.
- **Private channels**: Only existing members can access.
- **Default channel (#general)**: Cannot be archived. Cannot be made private.
- **DM/Group DM channels**: Visibility cannot be changed.

### Permission Functions (Go)

These are defined in `api/internal/channel/model.go`:

| Function             | Returns true for      | Used for                  |
| -------------------- | --------------------- | ------------------------- |
| `CanPost()`          | admin, poster, or nil | Sending messages          |
| `CanManageChannel()` | admin only            | Updating channel settings |

## Message Permissions

| Action     | Who can do it                                                                        |
| ---------- | ------------------------------------------------------------------------------------ |
| **Send**   | Channel members with posting permission; workspace members auto-join public channels |
| **Edit**   | Message author only. Cannot edit system messages or deleted messages.                |
| **Delete** | Message author OR workspace owner/admin                                              |

## File & Emoji Permissions

| Action                  | Who can do it                           |
| ----------------------- | --------------------------------------- |
| **Upload file**         | Any channel member who can post         |
| **Delete file**         | File uploader OR workspace owner/admin  |
| **Upload custom emoji** | Any workspace member                    |
| **Delete custom emoji** | Emoji uploader OR workspace owner/admin |

## Moderation

### Banning

Workspace owners and admins can ban members from a workspace. Banning removes the user's membership and prevents them from rejoining via invite links.

| Action         | Owner | Admin | Member | Guest |
| -------------- | :---: | :---: | :----: | :---: |
| Ban a user     |   ✓   |   ✓   |        |       |
| Unban a user   |   ✓   |   ✓   |        |       |
| View bans list |   ✓   |   ✓   |        |       |

**Restrictions**:

- Cannot ban yourself
- Cannot ban users with an equal or higher role (admins cannot ban other admins or the owner)
- Bans can be permanent or temporary (with expiry)
- Optional: hide the banned user's messages from other members

### Message Pinning

| Action          | Owner | Admin | Member | Guest |
| --------------- | :---: | :---: | :----: | :---: |
| Pin a message   |   ✓   |   ✓   |        |       |
| Unpin a message |   ✓   |   ✓   |        |       |
| View pinned     |   ✓   |   ✓   |   ✓    |   ✓   |

### Personal Blocking

Any workspace member can block another member within that workspace. Blocks are workspace-scoped and invisible to the blocked user.

| Action       | Owner | Admin | Member | Guest |
| ------------ | :---: | :---: | :----: | :---: |
| Block user   |   ✓   |   ✓   |   ✓    |   ✓   |
| Unblock user |   ✓   |   ✓   |   ✓    |   ✓   |
| List blocks  |   ✓   |   ✓   |   ✓    |   ✓   |

**Restrictions**:

- Cannot block yourself
- Cannot block users with `admin` or `owner` role in the workspace
- Blocks are workspace-scoped (blocking in workspace A does not affect workspace B)
- Blocks are one-directional (A blocking B does not prevent B from seeing A's messages)
- Unblocking has no role restriction — you can always undo your own block
- If a blocked user is later promoted to admin/owner, the existing block persists (role check applies at creation time only)

### Audit Log

All moderation actions are recorded in a workspace audit log.

| Action         | Owner | Admin | Member | Guest |
| -------------- | :---: | :---: | :----: | :---: |
| View audit log |   ✓   |   ✓   |        |       |

**Logged actions**: `user.banned`, `user.unbanned`, `member.removed`, `member.role_changed`, `message.deleted` (admin delete), `channel.archived`

## Server Level

There are no server-level roles. The `GET /api/server/info` endpoint returns only the server version and is unauthenticated. All administrative actions are scoped to individual workspaces.
