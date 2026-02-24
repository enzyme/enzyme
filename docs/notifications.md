# Notifications & Presence

This guide covers how notifications are triggered, delivered, and configured, as well as the real-time presence and typing indicator systems.

## Notification Preferences

Each channel has a per-user notification preference with two settings:

| Setting           | Values                    | Default (channels) | Default (DMs) |
| ----------------- | ------------------------- | ------------------ | ------------- |
| **Notify level**  | `all`, `mentions`, `none` | `mentions`         | `all`         |
| **Email enabled** | on / off                  | on                 | on            |

- **all** — notify on every message in the channel.
- **mentions** — notify only on @mentions and special mentions.
- **none** — muted. All notifications from this channel are suppressed.

Preferences are set per channel via the channel notification settings UI or the `POST /channels/{id}/notifications` endpoint.

## What Triggers Notifications

| Trigger                                        | Who is notified                | Respects "none" (muted)?   |
| ---------------------------------------------- | ------------------------------ | -------------------------- |
| [User mention](messages.md#mentions) (`@user`) | The mentioned user             | Yes                        |
| `@here`                                        | Online members of the channel  | Yes                        |
| `@channel`                                     | All members of the channel     | Yes                        |
| `@everyone`                                    | All members of the workspace   | Yes                        |
| DM / group DM message                          | All other participants         | N/A (DMs default to `all`) |
| Thread reply                                   | Users subscribed to the thread | No (overrides mute)        |

A user is auto-subscribed to a thread when they post a reply. The parent message author is also auto-subscribed when the first reply is posted. Users can explicitly unsubscribe from a thread, and auto-subscribe respects that choice.

## Email Notifications

When a user is offline and has `email_enabled` turned on for a channel, notifications are queued for email delivery.

**Delivery timing:**

1. A pending notification is created with a 5-minute delay.
2. A background worker checks for ready-to-send notifications every 60 seconds.
3. Notifications are grouped by workspace into a single digest email.
4. If the user comes back online before the delay expires, the pending notification is cancelled automatically.

Email notifications require SMTP to be configured. See [Email configuration](configuration.md#email) for setup.

## Presence

Enzyme tracks two presence states per user per workspace:

| State       | Meaning                           |
| ----------- | --------------------------------- |
| **online**  | User has an active SSE connection |
| **offline** | No SSE connection for 30 seconds  |

When a user disconnects (closes the tab, loses network), the server waits 30 seconds before marking them offline. This grace period absorbs brief interruptions like page refreshes.

Presence changes are broadcast to all connected workspace members via SSE. For SSE tuning options, see [SSE configuration](configuration.md#sse-real-time-events).

## Typing Indicators

Typing indicators are real-time and ephemeral — they are never persisted.

- When a user starts typing, a `typing.start` event is sent (throttled to once per 2 seconds).
- If the user stops typing for 3 seconds, a `typing.stop` event is sent automatically.
- On the receiving side, typing indicators expire after 5 seconds without a refresh.
