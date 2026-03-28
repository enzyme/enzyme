/**
 * Pure cache update functions for SSE events.
 *
 * These functions operate on a QueryClient and update the TanStack Query cache.
 * They contain NO platform-specific side effects (no browser notifications,
 * no audio, no toast, no navigation). Platform-specific behavior is added by
 * the consuming app's useSSE hook.
 */
import type { QueryClient } from '@tanstack/react-query';
import type {
  MessageListResult,
  MessageWithUser,
  ChannelWithMembership,
  CustomEmoji,
  SSEEvent,
} from '@enzyme/api-client';
import {
  authKeys,
  messageKeys,
  threadKeys,
  channelKeys,
  workspaceKeys,
  emojiKeys,
  scheduledMessageKeys,
  pinnedMessageKeys,
} from '../queryKeys';
import {
  addTypingUser,
  removeTypingUser,
  setUserPresence,
  setMultipleUserPresence,
} from '../stores/presenceStore';
import { getUrls } from '../cache/signedUrlCache';

type MessagePages = { pages: MessageListResult[]; pageParams: (string | undefined)[] };

// Extract SSE event data types by event type name
type EventDataOf<T extends SSEEvent['type']> = Extract<SSEEvent, { type: T }>['data'];

// --- Message Events ---

export function handleNewMessage(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'message.new'>,
) {
  // Pre-warm signed URL cache for any attachments
  if (data.attachments && data.attachments.length > 0) {
    const ids = data.attachments.map((a) => a.id);
    getUrls(ids).catch(() => {}); // fire-and-forget
  }

  // Thread replies go to thread cache, and optionally to channel if broadcast
  if (data.thread_parent_id) {
    // Check if this message was already processed (e.g., by optimistic update from current user)
    const threadData = queryClient.getQueryData<MessagePages>(
      threadKeys.detail(data.thread_parent_id),
    );
    const alreadyProcessed = threadData?.pages.some((page) =>
      page.messages.some((m) => m.id === data.id),
    );

    // Add to thread cache if not already there
    if (!alreadyProcessed) {
      queryClient.setQueryData(
        threadKeys.detail(data.thread_parent_id),
        (old: MessagePages | undefined) => {
          if (!old) return old;

          // Thread messages are ordered ASC (oldest first), so append to end
          const newPages = [...old.pages];
          const lastPageIndex = newPages.length - 1;
          if (newPages[lastPageIndex]) {
            newPages[lastPageIndex] = {
              ...newPages[lastPageIndex],
              messages: [...newPages[lastPageIndex].messages, data],
            };
          }
          return { ...old, pages: newPages };
        },
      );
    }

    // If also_send_to_channel, add to the channel message cache too
    if (data.also_send_to_channel) {
      queryClient.setQueryData(
        messageKeys.list(data.channel_id),
        (old: MessagePages | undefined) => {
          if (!old) return old;
          const exists = old.pages.some((page) => page.messages.some((m) => m.id === data.id));
          if (exists) return old;
          const newPages = [...old.pages];
          if (newPages[0]) {
            newPages[0] = {
              ...newPages[0],
              messages: [data, ...newPages[0].messages],
            };
          }
          return { ...old, pages: newPages };
        },
      );
    }

    // Always update thread_participants, but only increment reply_count if we added the message
    queryClient.setQueryData(messageKeys.list(data.channel_id), (old: MessagePages | undefined) => {
      if (!old) return old;

      return {
        ...old,
        pages: old.pages.map((page) => ({
          ...page,
          messages: page.messages.map((m) => {
            if (m.id !== data.thread_parent_id) return m;

            // Check if user should be added to thread_participants
            const participants = m.thread_participants || [];
            const shouldAddParticipant =
              data.user_id && !participants.some((p) => p.user_id === data.user_id);

            return {
              ...m,
              // Only increment reply_count if the message wasn't already processed
              reply_count: alreadyProcessed ? m.reply_count : (m.reply_count || 0) + 1,
              last_reply_at: data.created_at,
              thread_participants: shouldAddParticipant
                ? [
                    ...participants,
                    {
                      user_id: data.user_id!,
                      display_name: data.user_display_name,
                      avatar_url: data.user_avatar_url,
                    },
                  ]
                : participants,
            };
          }),
        })),
      };
    });

    // Invalidate threads list so unread count and thread order refresh
    queryClient.invalidateQueries({ queryKey: threadKeys.userThreads(workspaceId) });
  } else {
    // Regular channel message - add to channel messages
    queryClient.setQueryData(messageKeys.list(data.channel_id), (old: MessagePages | undefined) => {
      if (!old) return old;

      // Check if message already exists
      const exists = old.pages.some((page) => page.messages.some((m) => m.id === data.id));
      if (exists) return old;

      const newPages = [...old.pages];
      if (newPages[0]) {
        newPages[0] = {
          ...newPages[0],
          messages: [data, ...newPages[0].messages],
        };
      }
      return { ...old, pages: newPages };
    });
  }

  // Increment unread count for channels (for non-thread messages and broadcast thread replies from other users)
  if (!data.thread_parent_id || data.also_send_to_channel) {
    const authData = queryClient.getQueryData<{ user?: { id: string } }>(authKeys.me());
    const currentUserId = authData?.user?.id;

    // Don't increment unread for our own messages
    if (data.user_id !== currentUserId) {
      queryClient.setQueryData(
        channelKeys.list(workspaceId),
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) =>
              c.id === data.channel_id ? { ...c, unread_count: c.unread_count + 1 } : c,
            ),
          };
        },
      );

    }
  }
}

export function handleMessageUpdated(
  queryClient: QueryClient,
  data: EventDataOf<'message.updated'>,
) {
  queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    let changed = false;
    const pages = old.pages.map((page) => {
      if (!page.messages.some((m) => m.id === data.id)) return page;
      changed = true;
      return {
        ...page,
        messages: page.messages.map((m) => (m.id === data.id ? { ...m, ...data } : m)),
      };
    });
    return changed ? { ...old, pages } : old;
  });
}

export function handleMessageDeleted(
  queryClient: QueryClient,
  data: EventDataOf<'message.deleted'>,
) {
  const { id, thread_parent_id } = data;

  // Check if this delete was already handled locally (by useDeleteMessage)
  let alreadyHandled = false;
  if (thread_parent_id) {
    const threadData = queryClient.getQueryData<{ pages: MessageListResult[] }>(
      threadKeys.detail(thread_parent_id),
    );
    if (threadData) {
      alreadyHandled = !threadData.pages.some((page) => page.messages.some((m) => m.id === id));
    }
  }

  queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    return {
      ...old,
      pages: old.pages.map((page) => ({
        ...page,
        messages: page.messages
          .map((m) => {
            if (m.id !== id) return m;
            // Messages with replies: mark as deleted (keep in cache for placeholder)
            if (m.reply_count > 0) {
              return { ...m, deleted_at: new Date().toISOString() };
            }
            // Messages without replies: return null to filter out
            return null;
          })
          .filter((m): m is MessageWithUser => m !== null),
      })),
    };
  });

  // Also filter from thread caches
  queryClient.setQueriesData({ queryKey: threadKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    return {
      ...old,
      pages: old.pages.map((page) => ({
        ...page,
        messages: page.messages.filter((m) => m.id !== id),
      })),
    };
  });

  // Decrement parent's reply_count if this was a thread reply
  if (thread_parent_id && !alreadyHandled) {
    queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
      if (!old) return old;
      return {
        ...old,
        pages: old.pages.map((page) => ({
          ...page,
          messages: page.messages.map((m) => {
            if (m.id !== thread_parent_id) return m;
            return { ...m, reply_count: Math.max((m.reply_count || 0) - 1, 0) };
          }),
        })),
      };
    });
  }
}

// --- Reaction Events ---

export function handleReactionAdded(queryClient: QueryClient, data: EventDataOf<'reaction.added'>) {
  queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    return {
      ...old,
      pages: old.pages.map((page) => ({
        ...page,
        messages: page.messages.map((m) => {
          if (m.id !== data.message_id) return m;
          const reactions = m.reactions || [];
          // Avoid duplicates (check by user + emoji, not just ID, to handle optimistic updates)
          if (reactions.some((r) => r.user_id === data.user_id && r.emoji === data.emoji)) {
            // Replace temp reaction with real one
            return {
              ...m,
              reactions: reactions.map((r) =>
                r.user_id === data.user_id && r.emoji === data.emoji ? data : r,
              ),
            };
          }
          return { ...m, reactions: [...reactions, data] };
        }),
      })),
    };
  });
}

export function handleReactionRemoved(
  queryClient: QueryClient,
  data: EventDataOf<'reaction.removed'>,
) {
  const { message_id, user_id, emoji } = data;

  queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    return {
      ...old,
      pages: old.pages.map((page) => ({
        ...page,
        messages: page.messages.map((m) => {
          if (m.id !== message_id) return m;
          const reactions = m.reactions || [];
          return {
            ...m,
            reactions: reactions.filter((r) => !(r.user_id === user_id && r.emoji === emoji)),
          };
        }),
      })),
    };
  });
}

// --- Channel Events ---

export function handleChannelCreated(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'channel.created'>,
) {
  queryClient.setQueryData(
    channelKeys.list(workspaceId),
    (old: { channels: ChannelWithMembership[] } | undefined) => {
      if (!old) return old;
      // Avoid duplicates
      if (old.channels.some((c) => c.id === data.id)) return old;
      return {
        ...old,
        channels: [
          ...old.channels,
          { ...data, unread_count: 0, notification_count: 0, is_starred: false },
        ],
      };
    },
  );
}

export function handleChannelsInvalidate(queryClient: QueryClient, workspaceId: string) {
  queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) });
}

export function handleChannelUpdated(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'channel.updated'>,
) {
  queryClient.setQueryData(
    channelKeys.list(workspaceId),
    (old: { channels: ChannelWithMembership[] } | undefined) => {
      if (!old) return old;
      return {
        ...old,
        channels: old.channels
          .map((c) => (c.id === data.id ? { ...c, ...data } : c))
          .filter((c) => {
            // Remove channels that became private if user is not a member
            if (c.id === data.id && c.type === 'private' && !c.channel_role) {
              return false;
            }
            return true;
          }),
      };
    },
  );
}

export function handleChannelArchived(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'channel.archived'>,
) {
  queryClient.setQueryData(
    channelKeys.list(workspaceId),
    (old: { channels: ChannelWithMembership[] } | undefined) => {
      if (!old) return old;
      return {
        ...old,
        channels: old.channels.filter((c) => c.id !== data.id),
      };
    },
  );
}

export function handleMemberAdded(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'channel.member_added'>,
) {
  const { channel_id } = data;
  queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) });
  queryClient.invalidateQueries({ queryKey: channelKeys.members(channel_id) });
  queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
}

export function handleMemberRemoved(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'channel.member_removed'>,
) {
  const { channel_id } = data;
  queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) });
  queryClient.invalidateQueries({ queryKey: channelKeys.members(channel_id) });
  queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
}

export function handleChannelRead(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'channel.read'>,
) {
  const { channel_id, last_read_message_id } = data;
  queryClient.setQueryData(
    channelKeys.list(workspaceId),
    (old: { channels: ChannelWithMembership[] } | undefined) => {
      if (!old) return old;
      return {
        ...old,
        channels: old.channels.map((c) =>
          c.id === channel_id
            ? { ...c, unread_count: 0, notification_count: 0, last_read_message_id }
            : c,
        ),
      };
    },
  );
  queryClient.invalidateQueries({ queryKey: workspaceKeys.notifications() });
}

// --- Emoji Events ---

export function handleEmojiCreated(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'emoji.created'>,
) {
  queryClient.setQueryData(
    emojiKeys.list(workspaceId),
    (old: { emojis: CustomEmoji[] } | undefined) => {
      if (!old) return old;
      if (old.emojis.some((e) => e.id === data.id)) return old;
      return {
        ...old,
        emojis: [...old.emojis, data].sort((a, b) => a.name.localeCompare(b.name)),
      };
    },
  );
}

export function handleEmojiDeleted(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'emoji.deleted'>,
) {
  const { id } = data;
  queryClient.setQueryData(
    emojiKeys.list(workspaceId),
    (old: { emojis: CustomEmoji[] } | undefined) => {
      if (!old) return old;
      return { ...old, emojis: old.emojis.filter((e) => e.id !== id) };
    },
  );
}

// --- Workspace Events ---

export function handleWorkspaceUpdated(queryClient: QueryClient, workspaceId: string) {
  queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(workspaceId) });
}

// --- Scheduled Message Events ---

export function handleScheduledMessageChange(queryClient: QueryClient, workspaceId: string) {
  queryClient.invalidateQueries({ queryKey: scheduledMessageKeys.list(workspaceId) });
}

export function handleScheduledMessageSent(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'scheduled_message.sent'>,
) {
  queryClient.invalidateQueries({ queryKey: scheduledMessageKeys.list(workspaceId) });
  queryClient.invalidateQueries({ queryKey: messageKeys.list(data.channel_id) });
}

// --- Pin Events ---

export function handleMessagePinned(queryClient: QueryClient, data: EventDataOf<'message.pinned'>) {
  queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    let changed = false;
    const pages = old.pages.map((page) => {
      if (!page.messages.some((m) => m.id === data.id)) return page;
      changed = true;
      return {
        ...page,
        messages: page.messages.map((m) =>
          m.id === data.id ? { ...m, pinned_at: data.pinned_at, pinned_by: data.pinned_by } : m,
        ),
      };
    });
    return changed ? { ...old, pages } : old;
  });
  queryClient.invalidateQueries({ queryKey: pinnedMessageKeys.all });
}

export function handleMessageUnpinned(
  queryClient: QueryClient,
  data: EventDataOf<'message.unpinned'>,
) {
  queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    let changed = false;
    const pages = old.pages.map((page) => {
      if (!page.messages.some((m) => m.id === data.id)) return page;
      changed = true;
      return {
        ...page,
        messages: page.messages.map((m) =>
          m.id === data.id ? { ...m, pinned_at: undefined, pinned_by: undefined } : m,
        ),
      };
    });
    return changed ? { ...old, pages } : old;
  });
  queryClient.invalidateQueries({ queryKey: pinnedMessageKeys.all });
}

// --- Member Events ---

export function handleMemberBanned(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'member.banned'>,
) {
  queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
  queryClient.invalidateQueries({ queryKey: workspaceKeys.bans(workspaceId) });

  // If the current user was banned, refresh auth state to pick up the ban field
  const authData = queryClient.getQueryData<{ user?: { id: string } }>(authKeys.me());
  if (authData?.user?.id === data.user_id) {
    queryClient.invalidateQueries({ queryKey: authKeys.me() });
  }
}

export function handleMemberUnbanned(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'member.unbanned'>,
): boolean {
  queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
  queryClient.invalidateQueries({ queryKey: workspaceKeys.bans(workspaceId) });

  // If the current user was unbanned, refresh auth state to clear the ban field
  const authData = queryClient.getQueryData<{ user?: { id: string } }>(authKeys.me());
  const isCurrentUser = authData?.user?.id === data.user_id;
  if (isCurrentUser) {
    queryClient.invalidateQueries({ queryKey: authKeys.me() });
  }
  return isCurrentUser;
}

export function handleMemberLeft(queryClient: QueryClient, workspaceId: string) {
  queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
}

export function handleMemberRoleChanged(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'member.role_changed'>,
) {
  queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });

  // Only refresh auth if the current user's role changed
  const authData = queryClient.getQueryData<{ user?: { id: string } }>(authKeys.me());
  if (authData?.user?.id === data.user_id) {
    queryClient.invalidateQueries({ queryKey: authKeys.me() });
  }
}

// --- Typing & Presence Events ---

export function handleTypingStart(data: EventDataOf<'typing.start'>) {
  addTypingUser(data.channel_id, data);
}

export function handleTypingStop(data: EventDataOf<'typing.stop'>) {
  removeTypingUser(data.channel_id, data.user_id);
}

export function handlePresenceChanged(data: EventDataOf<'presence.changed'>) {
  setUserPresence(data.user_id, data.status);
}

export function handlePresenceInitial(data: EventDataOf<'presence.initial'>) {
  setMultipleUserPresence(data.online_user_ids.map((userId) => [userId, 'online']));
}

// --- Notification Events ---

/**
 * Updates notification count in cache. Returns the notification data
 * for platform-specific handling (sound, browser notification, etc.).
 */
export function handleNotification(
  queryClient: QueryClient,
  workspaceId: string,
  data: EventDataOf<'notification'>,
): EventDataOf<'notification'> {
  // Increment notification_count for the channel
  if (data.channel_id && data.type !== 'thread_reply') {
    queryClient.setQueryData(
      channelKeys.list(workspaceId),
      (old: { channels: ChannelWithMembership[] } | undefined) => {
        if (!old) return old;
        return {
          ...old,
          channels: old.channels.map((c) =>
            c.id === data.channel_id ? { ...c, notification_count: c.notification_count + 1 } : c,
          ),
        };
      },
    );
  }

  return data;
}
