---
title: 'Sending Messages'
description: 'Compose, edit, delete, and schedule messages'
section: 'Using Enzyme'
order: 14
---

# Sending Messages

This guide covers composing, editing, deleting, and scheduling messages.

## Composing

Click the message input at the bottom of a channel or start typing anywhere to auto-focus the composer. Press **Enter** to send, or **Shift+Enter** for a new line.

While typing, you can use autocomplete:

- Type `@` to mention a user
- Type `#` to reference a channel
- Type `:` to insert an emoji

See [Message Formatting](/docs/messages/) for the full formatting reference.

## File Attachments

Click the attachment button in the composer or drag and drop files into the message area. Multiple files can be attached to a single message.

- Maximum file size: **10 MB** (configurable by the server admin)
- All file types are accepted
- Images display inline with previews

See [Message Formatting — Attachments](/docs/messages/#attachments) for image layout details.

## Editing Messages

To edit a message you've sent, hover over it and click the edit button (pencil icon) in the message actions menu. Admins and owners can edit any message. Edited messages show an "(edited)" indicator.

The same [character limit](/docs/messages/#character-limit) (40,000 characters) applies to edits.

## Deleting Messages

To delete a message, hover over it and click the delete button (trash icon) in the message actions menu.

- Messages **with replies** are marked as deleted but remain as a placeholder so the thread stays intact.
- Messages **without replies** are fully removed.
- Admins and owners can delete any message.

## Scheduled Messages

You can schedule a message to be sent at a future time:

1. Click the clock icon next to the send button.
2. Choose a date and time.
3. The message is saved and will be sent automatically at the scheduled time.

### Managing Scheduled Messages

Access your scheduled messages from the sidebar or via the command palette (search "Scheduled messages"). From there you can:

- **Edit** the content or scheduled time
- **Delete** a scheduled message before it sends
- **Send immediately** if you don't want to wait

If a scheduled message fails to send (e.g., you were removed from the channel), you'll receive a notification.

## Typing Indicators

When you're composing a message, other users in the channel see a typing indicator. These are real-time and ephemeral — they appear while you type and disappear shortly after you stop.
