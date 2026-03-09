---
title: 'Reactions & Emoji'
description: 'Emoji reactions and custom emoji'
section: 'Using Enzyme'
order: 17
---

# Reactions & Emoji

React to messages with emoji and upload custom emoji for your workspace.

## Adding Reactions

Hover over a message and click the smiley face icon to open the emoji picker, then select an emoji. You can also click an existing reaction to add yours to the same emoji.

Multiple people can react with the same emoji — the reaction shows a count. Click your own reaction again to remove it.

## Emoji in Messages

Use `:shortcode:` syntax to insert emoji in messages. For example, `:rocket:` renders as a rocket emoji. Enzyme supports the standard [GitHub gemoji](https://github.com/github/gemoji) shortcode set.

You can also use the autocomplete — type `:` followed by a few characters to search for emoji by name.

### Large Emoji

Messages containing **only 1-3 emoji** (with no other text) render at 4x size.

## Custom Emoji

Workspace members can upload custom emoji that everyone in the workspace can use.

### Uploading

1. Open workspace settings.
2. Navigate to the custom emoji section.
3. Upload an image and give it a name.

**Requirements:**

| Property   | Constraint                                               |
| ---------- | -------------------------------------------------------- |
| **Format** | PNG or GIF only                                          |
| **Size**   | Max 256 KB                                               |
| **Name**   | Alphanumeric, hyphens, and underscores (1-63 characters) |

Custom emoji use the same `:shortcode:` syntax as standard emoji. If a custom emoji has the same name as a built-in emoji, the built-in one takes precedence.

### Deleting

Custom emoji can be deleted by the user who uploaded them, or by workspace admins and owners. Deleting a custom emoji removes it from the picker but doesn't affect messages that already used it.
