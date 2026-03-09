---
title: 'Search'
description: 'Search messages across your workspace'
section: 'Using Enzyme'
order: 19
---

# Search

Enzyme provides full-text search across all messages in your workspace.

## Opening Search

- Press **Cmd+Shift+F** (Mac) or **Ctrl+Shift+F** (Windows/Linux), or
- Open the command palette (**Cmd+K**) and type your search query — select "Search messages for..." to run a full search.

## Search Filters

Narrow your results using filters:

| Filter      | Description                      |
| ----------- | -------------------------------- |
| **Channel** | Search within a specific channel |
| **From**    | Messages sent by a specific user |
| **Before**  | Messages sent before a date      |
| **After**   | Messages sent after a date       |

Filters can be combined. For example, search for messages from a specific person in a specific channel within a date range.

## Search Scope

Search covers all channels you have access to:

- **Public channels** — searchable even if you haven't joined
- **Private channels** — only searchable if you're a member
- **DMs and group DMs** — searchable

Search results show the matching message with its channel, author, and timestamp. Click a result to jump to that message in context.

## Full-Text Search

Search uses SQLite's FTS5 full-text search engine. It supports:

- **Partial words** — searching "deploy" matches "deployment" and "deploying"
- **Multiple terms** — all terms must appear in the message
- **Case-insensitive** matching
