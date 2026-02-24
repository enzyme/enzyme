# Workspace Administration

This guide covers workspace settings, invites, member management, and channel administration. See [Permissions & Roles](permissions.md) for the full permission matrix.

## Workspace Settings

Owners and admins can update the following workspace settings:

| Setting                      | Description                                                              |
| ---------------------------- | ------------------------------------------------------------------------ |
| **Name**                     | Workspace display name                                                   |
| **Icon**                     | JPEG, PNG, GIF, or WebP image (max 5 MB)                                 |
| **Show join/leave messages** | Toggle system messages when members join or leave channels (default: on) |

## Invites

Owners and admins can create invite links to add new members.

**Invite properties:**

| Property     | Details                                                 |
| ------------ | ------------------------------------------------------- |
| **Code**     | 32-character random hex string, generated automatically |
| **Role**     | `admin`, `member`, or `guest` (cannot invite as owner)  |
| **Expiry**   | Optional, specified in hours from creation              |
| **Max uses** | Optional use limit; the invite is rejected once reached |

**What happens when an invite is accepted:**

1. The user joins the workspace with the role specified in the invite.
2. They are automatically added to the #general channel.
3. DMs are auto-created with up to 5 existing workspace members (earliest joined first).

To send invites via email, configure SMTP first. See [Email configuration](configuration.md#email).

## Managing Members

Owners and admins can list members, change roles, and remove members. Key rules:

- The **owner cannot be removed** or have their role changed. Ownership is permanent and non-transferable.
- **Admins cannot promote** other members to admin — only the owner can.
- Admins can assign the `member` or `guest` roles.
- Any member can **remove themselves** from a workspace (except the owner).
- Users **cannot change their own role**.

See [Permissions & Roles](permissions.md) for the complete permission matrix and special rules.

## Channels

### Creating Channels

Members, admins, and owners can create channels. Guests cannot.

Channel names must match the pattern `^[a-z0-9]+(-[a-z0-9]+)*$` — lowercase alphanumeric characters separated by single hyphens. Names cannot start or end with a hyphen, and consecutive hyphens are not allowed. There is no explicit length limit.

### Public vs Private

- **Public channels** are visible to all workspace members. Non-members who post are auto-joined.
- **Private channels** are visible only to their members. Members must be added explicitly.

Visibility can be changed between public and private by channel admins or workspace owners/admins.

### Channel Roles

Channels have their own role system independent of workspace roles:

| Role       | Can post | Can manage channel |
| ---------- | :------: | :----------------: |
| **admin**  |   yes    |        yes         |
| **poster** |   yes    |                    |
| **viewer** |          |                    |

See [Channel Roles](permissions.md#channel-roles) for details.

### Archiving

Workspace owners and admins can archive channels. Archived channels become read-only. DM/group DM channels and the default channel cannot be archived.

## Default Channel (#general)

Every workspace has a #general channel created automatically. It has special rules:

- New members are auto-joined on workspace entry.
- It cannot be archived.
- It cannot be made private.

## DMs & Group DMs

DMs and group DMs are created automatically when users start a conversation.

- **DMs** have exactly 2 participants.
- **Group DMs** have 3-8 participants.
- Visibility cannot be changed (they are always private to participants).
- They cannot be archived.
- Group DMs can be **converted to channels** by members, admins, or owners (guests cannot). See [Permission Matrix](permissions.md#permission-matrix). This gives the channel a name and makes it appear in the channel list.
