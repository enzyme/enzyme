---
title: 'Reading Messages'
description: 'Unreads, mark as read, and message navigation'
section: 'Using Enzyme'
order: 15
---

# Reading Messages

This guide covers how to navigate messages, manage unreads, and interact with message content.

## Unreads

Channels with unread messages are highlighted in the sidebar. The badge shows the number of unread notifications (mentions and DMs), not total unread messages.

### All Unreads View

The **All Unreads** page shows unread messages across all your channels in one place. Access it from the sidebar or via the command palette.

### Marking as Read

- **Single channel** — Open a channel to automatically mark it as read, or right-click it in the sidebar to mark as read without opening.
- **All channels** — Use the sidebar menu to mark all channels in the workspace as read.
- **Mark as unread** — Hover over a message and select "Mark unread" from the message actions menu to set a manual unread marker from that point.

## Message Actions

Hover over any message to reveal the action buttons:

| Action              | Description                                     |
| ------------------- | ----------------------------------------------- |
| **React**           | Add an emoji reaction                           |
| **Reply in thread** | Start or continue a thread                      |
| **Pin / Unpin**     | Pin the message to the channel                  |
| **Edit**            | Edit your own message (admins can edit any)     |
| **Delete**          | Delete your own message (admins can delete any) |
| **Mark unread**     | Mark as unread from this message                |

## Link Previews

The first URL in a message automatically generates a preview with the page's title, description, and image (via Open Graph metadata). Previews are cached for 24 hours.

Internal message links (links to other messages within Enzyme) show an inline preview of the referenced message, including the author, content, timestamp, and channel. If you don't have access to the referenced channel, the content is redacted.

## Image Viewer

Click any image attachment to open a full-screen viewer. Use arrow keys to navigate between images in the message. A download button is available in the viewer.
