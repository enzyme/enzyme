---
title: 'Permissions & Roles'
description: 'Role-based access control and permission matrix'
section: 'Administration'
order: 32
---

# Permissions & Roles

Enzyme uses role-based access control at two levels: **workspace** and **channel**. There are no server-level admin roles — all permissions are scoped to individual workspaces.

## Workspace Roles

Every workspace member has exactly one role. Roles are ordered by privilege:

| Role       | Rank | Description                               |
| ---------- | ---- | ----------------------------------------- |
| **Owner**  | 4    | Highest privilege, full workspace control |
| **Admin**  | 3    | Can manage members and workspace settings |
| **Member** | 2    | Standard participant                      |
| **Guest**  | 1    | Limited access, restricted by default     |

See [Workspace Administration](/docs/administration/) for the operational guide on managing members, invites, and settings.

### Permission Matrix

| Action                              | Owner | Admin | Member | Guest |
| ----------------------------------- | :---: | :---: | :----: | :---: |
| Post messages (in allowed channels) |   ✓   |   ✓   |   ✓    |   ✓   |
| React to messages                   |   ✓   |   ✓   |   ✓    |   ✓   |
| Edit own messages                   |   ✓   |   ✓   |   ✓    |   ✓   |
| Delete own messages                 |   ✓   |   ✓   |   ✓    |   ✓   |
| Delete own custom emoji             |   ✓   |   ✓   |   ✓    |   ✓   |
| Create channels ⚙                   |   ✓   |   ✓   |   ✓    |       |
| Convert group DM to channel ⚙       |   ✓   |   ✓   |   ✓    |       |
| Upload custom emoji ⚙               |   ✓   |   ✓   |   ✓    |       |
| Pin/unpin messages ⚙                |   ✓   |   ✓   |   ✓    |       |
| Create invite links ⚙               |   ✓   |   ✓   |        |       |
| Delete any message                  |   ✓   |   ✓   |        |       |
| Delete any custom emoji             |   ✓   |   ✓   |        |       |
| Delete any file                     |   ✓   |   ✓   |        |       |
| Manage members (add/remove)         |   ✓   |   ✓   |        |       |
| Change member roles                 |   ✓   |   ✓   |        |       |
| Update workspace name/settings      |   ✓   |   ✓   |        |       |
| Upload/remove workspace icon        |   ✓   |   ✓   |        |       |
| Archive channels                    |   ✓   |   ✓   |        |       |
| Promote member to admin             |   ✓   |       |        |       |
| Promote member to owner             |   ✓   |       |        |       |
| Delete workspace                    |   ✓   |       |        |       |

Actions marked with ⚙ are **configurable** — see [Configurable Permission Settings](#configurable-permission-settings) below.

### Special Rules

- **Multiple owners**: A workspace can have multiple owners. Any owner can promote any member to owner.
- **Owners cannot demote other owners**: An owner can only demote themselves, not another owner.
- **Last owner protection**: The last remaining owner cannot demote themselves or leave the workspace. At least one owner must always exist.
- **Admins cannot promote to admin**: Only an owner can make someone an admin. Admins can only assign the "member" role.
- **Self-removal**: Any member can leave a workspace. Owners can leave if at least one other owner exists.

### Configurable Permission Settings

Workspace owners and admins can adjust who can perform certain actions via the **Permissions** tab in workspace settings. Each setting accepts one of three levels:

| Level        | Includes                    |
| ------------ | --------------------------- |
| **Everyone** | Owner, Admin, Member, Guest |
| **Members**  | Owner, Admin, Member        |
| **Admins**   | Owner, Admin                |

The four configurable permissions and their defaults:

| Setting                         | Default | Controls                                                                |
| ------------------------------- | ------- | ----------------------------------------------------------------------- |
| **Who can create channels**     | Members | Creating channels, converting group DMs to channels                     |
| **Who can create invites**      | Admins  | Generating workspace invite links                                       |
| **Who can pin messages**        | Members | Pinning and unpinning messages                                          |
| **Who can manage custom emoji** | Members | Uploading custom emoji (deletion of others' emoji is always admin-only) |

Changes take effect immediately. Existing workspaces that haven't configured these settings use the defaults, preserving backward compatibility.

### Permission Functions (Go)

These are defined in `api/internal/workspace/model.go`:

| Function               | Returns true for | Used for                                              |
| ---------------------- | ---------------- | ----------------------------------------------------- |
| `CanManageMembers()`   | Owner, Admin     | Add/remove members, settings, icon, archive channels  |
| `CanChangeRole()`      | Owner, Admin     | Change member roles (with additional restrictions)    |
| `HasPermission()`      | Depends on level | Configurable actions (channels, invites, pins, emoji) |
| `CanDeleteWorkspace()` | Owner            | Delete workspace (not yet implemented in handler)     |

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

| Action                  | Who can do it                                                            |
| ----------------------- | ------------------------------------------------------------------------ |
| **Upload file**         | Any channel member who can post                                          |
| **Delete file**         | File uploader OR workspace owner/admin                                   |
| **Upload custom emoji** | Controlled by **who can manage custom emoji** setting (default: members) |
| **Delete custom emoji** | Emoji uploader OR workspace owner/admin                                  |

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

Pinning is controlled by the **who can pin messages** workspace setting (default: members).

| Action          | Default permission       |
| --------------- | ------------------------ |
| Pin a message   | Members (configurable ⚙) |
| Unpin a message | Members (configurable ⚙) |
| View pinned     | All channel members      |

The user must also be a channel member who can post (or a workspace admin for public channels).

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
