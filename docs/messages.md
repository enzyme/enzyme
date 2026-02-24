# Message Formatting

This guide covers Enzyme's message formatting syntax, mentions, links, attachments, and related features.

## mrkdwn Syntax

Enzyme uses **mrkdwn**, a Slack-compatible markup language for formatting messages.

### Inline Formatting

| Syntax              | Result            |
| ------------------- | ----------------- |
| `*bold*`            | **bold**          |
| `_italic_`          | _italic_          |
| `~strikethrough~`   | ~~strikethrough~~ |
| `` `inline code` `` | `inline code`     |

Unclosed formatting markers are rendered as literal text (e.g., `*unclosed` stays as `*unclosed`).

### Code Blocks

Wrap code in triple backticks with an optional language identifier:

````
```js
const x = 42;
```
````

### Blockquotes

Prefix lines with `>`:

```
> This is a blockquote
> spanning multiple lines
```

Each line must start with `>`. Content inside blockquotes supports inline formatting.

### Lists

**Bullet lists** use `•` or `-`:

```
- First item
- Second item
- Third item
```

**Ordered lists** use `1.`, `2.`, etc.:

```
1. First item
2. Second item
3. Third item
```

List item content supports inline formatting.

## Links

Links use angle bracket syntax with `|` separating the URL from the display text:

```
<https://example.com>
<https://example.com|Example>
```

The first form displays the raw URL. The second displays "Example" as a clickable link. URLs must include a scheme (`http://` or `https://`).

### Link Previews

The first URL in a message automatically generates a link preview.

**External links** fetch Open Graph metadata (title, description, image) from the target page. Results are cached for 24 hours (1 hour on fetch error). If metadata isn't cached yet, the preview is fetched asynchronously and broadcast to clients via SSE when ready.

**Internal message links** — URLs matching the pattern `/workspaces/{id}/channels/{id}?msg={id}` — display an inline preview showing the referenced message's author, content (truncated to 300 characters), timestamp, and channel name. These previews respect access controls: if the viewer doesn't have access to the referenced channel, the content is redacted.

## Mentions

### User Mentions

Syntax: `<@userId>`

Renders as a clickable blue badge showing `@Display Name`. Clicking opens a profile popover with a "View profile" button. Triggers a notification for the mentioned user. See [Notifications](notifications.md) for notification preferences and delivery behavior.

When composing via the API, use the user's ULID (not their display name).

### Channel References

Syntax: `<#channelId>`

Renders as a clickable badge showing `#channel-name` with a hashtag icon (or a lock icon for private channels). Clicking opens a popover with the channel description and a "Go to channel" button. If the viewer doesn't have access to the channel, it renders as a generic "private-channel" with a lock icon.

When composing via the API, use the channel's ULID (not its name).

### Special Mentions

| Syntax        | Behavior                                                 |
| ------------- | -------------------------------------------------------- |
| `<!here>`     | Notifies everyone who is currently online in the channel |
| `<!channel>`  | Notifies everyone in the channel                         |
| `<!everyone>` | Notifies everyone in the workspace                       |

These render as highlighted `@here`, `@channel`, and `@everyone` badges.

## Emoji

### Shortcodes

Use `:shortcode:` syntax to insert emoji (e.g., `:smile:`, `:rocket:`, `:thumbsup:`). Standard shortcodes follow the [GitHub gemoji](https://github.com/github/gemoji) set. Unrecognized shortcodes are rendered as literal text.

### Custom Emoji

Workspaces can upload custom emoji (PNG or GIF, max 256 KB). Custom emoji use the same `:shortcode:` syntax. Standard emoji take precedence over custom emoji when names conflict.

- Name: alphanumeric characters, hyphens, and underscores (1-63 characters)
- Formats: PNG and GIF only

### Large Emoji

Messages containing only 1-3 emoji (with no other content) render at 4x size.

## Character Limit

Messages have a maximum length of **40,000 characters** (counted as UTF-8 runes, not bytes). This limit is enforced server-side when sending, editing, and scheduling messages. Exceeding it returns a `400` response with a validation error.

## Attachments

### File Uploads

- Maximum file size: **10 MB** (configurable via [`files.max_upload_size`](configuration.md#file-storage))
- All file types are accepted
- Multiple files can be attached to a single message

### Image Display

Images (PNG, JPEG, GIF, WebP) display inline with previews. Layout depends on the number of images:

| Count | Layout                                              |
| ----- | --------------------------------------------------- |
| 1     | Full-width thumbnail                                |
| 2     | Side by side (2-column grid)                        |
| 3     | First image full-width, two square thumbnails below |
| 4+    | 2x2 grid with a "+N" overlay on the last cell       |

Clicking an image opens a full-screen carousel with arrow key navigation and a download button.

Non-image files display as a list with file name, size, and a download link.

### Other Upload Limits

| Type           | Limit  |
| -------------- | ------ |
| Custom emoji   | 256 KB |
| User avatar    | 5 MB   |
| Workspace icon | 5 MB   |

## System Messages

System messages are generated automatically for channel events. They cannot be edited, deleted, or sent via the API.

| Event                         | Description                               |
| ----------------------------- | ----------------------------------------- |
| `user_joined`                 | A user joined the channel                 |
| `user_left`                   | A user left the channel                   |
| `user_added`                  | An admin added a user to the channel      |
| `user_converted_channel`      | A group DM was converted to a channel     |
| `channel_renamed`             | The channel was renamed                   |
| `channel_visibility_changed`  | The channel type changed (public/private) |
| `channel_description_updated` | The channel description was updated       |
