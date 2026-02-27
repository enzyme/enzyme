import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { SSEConnection } from '../lib/sse';
import { getUrls } from '../lib/signedUrlCache';
import { addTypingUser, removeTypingUser, setUserPresence } from '../lib/presenceStore';
import {
  playNotificationSound,
  showBrowserNotification,
  unlockAudio,
  requestNotificationPermission,
} from '../lib/notificationSound';
import { toast } from '../components/ui';
import type {
  MessageListResult,
  MessageWithUser,
  ChannelWithMembership,
  NotificationData,
  CustomEmoji,
} from '@enzyme/api-client';

export function useSSE(workspaceId: string | undefined) {
  const [isConnected, setIsConnected] = useState(false);
  const [hasBeenConnected, setHasBeenConnected] = useState(false);
  const connectionRef = useRef<SSEConnection | null>(null);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  useEffect(() => {
    navigateRef.current = navigate;
  }, [navigate]);

  useEffect(() => {
    if (!workspaceId) {
      return;
    }

    const connection = new SSEConnection(workspaceId);
    connectionRef.current = connection;

    // Handle disconnect
    connection.setOnDisconnect(() => {
      setIsConnected(false);
    });

    // Handle 403 — stop reconnecting and refresh auth state
    connection.setOnForbidden(() => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    });

    // Handle connected event
    connection.on('connected', () => {
      setHasBeenConnected(true);
      setIsConnected(true);
    });

    // Handle new message
    connection.on('message.new', (event) => {
      const message = event.data;

      // Pre-warm signed URL cache for any attachments
      if (message.attachments && message.attachments.length > 0) {
        const ids = message.attachments.map((a) => a.id);
        getUrls(ids).catch(() => {}); // fire-and-forget
      }

      // Thread replies go to thread cache, and optionally to channel if broadcast
      if (message.thread_parent_id) {
        // Check if this message was already processed (e.g., by optimistic update from current user)
        const threadData = queryClient.getQueryData<{
          pages: MessageListResult[];
          pageParams: (string | undefined)[];
        }>(['thread', message.thread_parent_id]);
        const alreadyProcessed = threadData?.pages.some((page) =>
          page.messages.some((m) => m.id === message.id),
        );

        // Add to thread cache if not already there
        if (!alreadyProcessed) {
          queryClient.setQueryData(
            ['thread', message.thread_parent_id],
            (
              old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined,
            ) => {
              if (!old) return old;

              // Thread messages are ordered ASC (oldest first), so append to end
              const newPages = [...old.pages];
              const lastPageIndex = newPages.length - 1;
              if (newPages[lastPageIndex]) {
                newPages[lastPageIndex] = {
                  ...newPages[lastPageIndex],
                  messages: [...newPages[lastPageIndex].messages, message],
                };
              }
              return { ...old, pages: newPages };
            },
          );
        }

        // If also_send_to_channel, add to the channel message cache too
        if (message.also_send_to_channel) {
          queryClient.setQueryData(
            ['messages', message.channel_id],
            (
              old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined,
            ) => {
              if (!old) return old;
              const exists = old.pages.some((page) =>
                page.messages.some((m) => m.id === message.id),
              );
              if (exists) return old;
              const newPages = [...old.pages];
              if (newPages[0]) {
                newPages[0] = {
                  ...newPages[0],
                  messages: [message, ...newPages[0].messages],
                };
              }
              return { ...old, pages: newPages };
            },
          );
        }

        // Always update thread_participants, but only increment reply_count if we added the message
        // (useSendThreadReply already increments reply_count for the sender's own messages)
        queryClient.setQueryData(
          ['messages', message.channel_id],
          (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
            if (!old) return old;

            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                messages: page.messages.map((m) => {
                  if (m.id !== message.thread_parent_id) return m;

                  // Check if user should be added to thread_participants
                  const participants = m.thread_participants || [];
                  const shouldAddParticipant =
                    message.user_id && !participants.some((p) => p.user_id === message.user_id);

                  return {
                    ...m,
                    // Only increment reply_count if the message wasn't already processed
                    reply_count: alreadyProcessed ? m.reply_count : (m.reply_count || 0) + 1,
                    last_reply_at: message.created_at,
                    thread_participants: shouldAddParticipant
                      ? [
                          ...participants,
                          {
                            user_id: message.user_id!,
                            display_name: message.user_display_name,
                            avatar_url: message.user_avatar_url,
                          },
                        ]
                      : participants,
                  };
                }),
              })),
            };
          },
        );

        // Invalidate threads list so unread count and thread order refresh
        queryClient.invalidateQueries({ queryKey: ['user-threads'] });
      } else {
        // Regular channel message - add to channel messages
        queryClient.setQueryData(
          ['messages', message.channel_id],
          (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
            if (!old) return old;

            // Check if message already exists
            const exists = old.pages.some((page) => page.messages.some((m) => m.id === message.id));
            if (exists) return old;

            const newPages = [...old.pages];
            if (newPages[0]) {
              newPages[0] = {
                ...newPages[0],
                messages: [message, ...newPages[0].messages],
              };
            }
            return { ...old, pages: newPages };
          },
        );
      }

      // Increment unread count for channels (for non-thread messages and broadcast thread replies from other users)
      if (!message.thread_parent_id || message.also_send_to_channel) {
        const authData = queryClient.getQueryData<{ user?: { id: string } }>(['auth', 'me']);
        const currentUserId = authData?.user?.id;

        // Don't increment unread for our own messages
        if (message.user_id !== currentUserId) {
          queryClient.setQueryData(
            ['channels', workspaceId],
            (old: { channels: ChannelWithMembership[] } | undefined) => {
              if (!old) return old;
              return {
                ...old,
                channels: old.channels.map((c) =>
                  c.id === message.channel_id ? { ...c, unread_count: c.unread_count + 1 } : c,
                ),
              };
            },
          );

          // Also invalidate workspace notifications so non-active workspace indicators update
          queryClient.invalidateQueries({ queryKey: ['workspaces', 'notifications'] });
        }
      }
    });

    // Handle message updated
    connection.on('message.updated', (event) => {
      const message = event.data;

      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          let changed = false;
          const pages = old.pages.map((page) => {
            if (!page.messages.some((m) => m.id === message.id)) return page;
            changed = true;
            return {
              ...page,
              messages: page.messages.map((m) => (m.id === message.id ? { ...m, ...message } : m)),
            };
          });
          return changed ? { ...old, pages } : old;
        },
      );
    });

    // Handle message deleted
    connection.on('message.deleted', (event) => {
      const { id, thread_parent_id } = event.data;

      // Check if this delete was already handled locally (by useDeleteMessage)
      // by seeing if the message still exists in the thread cache
      let alreadyHandled = false;
      if (thread_parent_id) {
        const threadData = queryClient.getQueryData<{ pages: MessageListResult[] }>([
          'thread',
          thread_parent_id,
        ]);
        if (threadData) {
          alreadyHandled = !threadData.pages.some((page) => page.messages.some((m) => m.id === id));
        }
      }

      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
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
        },
      );

      // Also filter from thread caches
      queryClient.setQueriesData(
        { queryKey: ['thread'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.filter((m) => m.id !== id),
            })),
          };
        },
      );

      // Decrement parent's reply_count if this was a thread reply
      // Skip if useDeleteMessage already handled this (to avoid double-decrement)
      if (thread_parent_id && !alreadyHandled) {
        queryClient.setQueriesData(
          { queryKey: ['messages'] },
          (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
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
          },
        );
      }
    });

    // Handle reaction added
    connection.on('reaction.added', (event) => {
      const reaction = event.data;

      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((m) => {
                if (m.id !== reaction.message_id) return m;
                const reactions = m.reactions || [];
                // Avoid duplicates (check by user + emoji, not just ID, to handle optimistic updates)
                if (
                  reactions.some(
                    (r) => r.user_id === reaction.user_id && r.emoji === reaction.emoji,
                  )
                ) {
                  // Replace temp reaction with real one
                  return {
                    ...m,
                    reactions: reactions.map((r) =>
                      r.user_id === reaction.user_id && r.emoji === reaction.emoji ? reaction : r,
                    ),
                  };
                }
                return { ...m, reactions: [...reactions, reaction] };
              }),
            })),
          };
        },
      );
    });

    // Handle reaction removed
    connection.on('reaction.removed', (event) => {
      const { message_id, user_id, emoji } = event.data;

      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
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
        },
      );
    });

    // Handle channel events
    connection.on('channel.created', () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    });

    connection.on('channel.updated', (event) => {
      const channel = event.data;
      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels
              .map((c) => (c.id === channel.id ? { ...c, ...channel } : c))
              .filter((c) => {
                // Remove channels that became private if user is not a member
                if (c.id === channel.id && c.type === 'private' && !c.channel_role) {
                  return false;
                }
                return true;
              }),
          };
        },
      );
    });

    connection.on('channel.archived', () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    });

    connection.on('channel.member_added', (event) => {
      const { channel_id } = event.data;
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
      queryClient.invalidateQueries({ queryKey: ['channel', channel_id, 'members'] });
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'members'] });
    });

    connection.on('channel.member_removed', (event) => {
      const { channel_id } = event.data;
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
      queryClient.invalidateQueries({ queryKey: ['channel', channel_id, 'members'] });
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'members'] });
    });

    // Handle channel read events (for syncing across tabs/devices)
    connection.on('channel.read', (event) => {
      const { channel_id, last_read_message_id } = event.data;
      queryClient.setQueryData(
        ['channels', workspaceId],
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
      queryClient.invalidateQueries({ queryKey: ['workspaces', 'notifications'] });
    });

    // Handle custom emoji events
    connection.on('emoji.created', (event) => {
      const emoji = event.data as CustomEmoji;
      queryClient.setQueryData(
        ['custom-emojis', workspaceId],
        (old: { emojis: CustomEmoji[] } | undefined) => {
          if (!old) return old;
          // Avoid duplicates
          if (old.emojis.some((e) => e.id === emoji.id)) return old;
          return {
            ...old,
            emojis: [...old.emojis, emoji].sort((a, b) => a.name.localeCompare(b.name)),
          };
        },
      );
    });

    connection.on('emoji.deleted', (event) => {
      const { id } = event.data as { id: string; name: string };
      queryClient.setQueryData(
        ['custom-emojis', workspaceId],
        (old: { emojis: CustomEmoji[] } | undefined) => {
          if (!old) return old;
          return { ...old, emojis: old.emojis.filter((e) => e.id !== id) };
        },
      );
    });

    // Handle scheduled message events
    connection.on('scheduled_message.created', () => {
      queryClient.invalidateQueries({ queryKey: ['scheduled-messages', workspaceId] });
    });
    connection.on('scheduled_message.updated', () => {
      queryClient.invalidateQueries({ queryKey: ['scheduled-messages', workspaceId] });
    });
    connection.on('scheduled_message.deleted', () => {
      queryClient.invalidateQueries({ queryKey: ['scheduled-messages', workspaceId] });
    });
    connection.on('scheduled_message.sent', (event) => {
      queryClient.invalidateQueries({ queryKey: ['scheduled-messages', workspaceId] });
      // Also invalidate the relevant channel messages
      if (event.data?.channel_id) {
        queryClient.invalidateQueries({ queryKey: ['messages', event.data.channel_id] });
      }
    });
    connection.on('scheduled_message.failed', () => {
      queryClient.invalidateQueries({ queryKey: ['scheduled-messages', workspaceId] });
      toast('A scheduled message failed to send', 'error');
    });

    // Handle message pinned
    connection.on('message.pinned', (event) => {
      const message = event.data;
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          let changed = false;
          const pages = old.pages.map((page) => {
            if (!page.messages.some((m) => m.id === message.id)) return page;
            changed = true;
            return {
              ...page,
              messages: page.messages.map((m) =>
                m.id === message.id
                  ? { ...m, pinned_at: message.pinned_at, pinned_by: message.pinned_by }
                  : m,
              ),
            };
          });
          return changed ? { ...old, pages } : old;
        },
      );
      queryClient.invalidateQueries({ queryKey: ['pinned-messages'] });
    });

    // Handle message unpinned
    connection.on('message.unpinned', (event) => {
      const message = event.data;
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          let changed = false;
          const pages = old.pages.map((page) => {
            if (!page.messages.some((m) => m.id === message.id)) return page;
            changed = true;
            return {
              ...page,
              messages: page.messages.map((m) =>
                m.id === message.id ? { ...m, pinned_at: undefined, pinned_by: undefined } : m,
              ),
            };
          });
          return changed ? { ...old, pages } : old;
        },
      );
      queryClient.invalidateQueries({ queryKey: ['pinned-messages'] });
    });

    // Handle member banned
    connection.on('member.banned', (event) => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'members'] });
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'bans'] });

      // If the current user was banned, refresh auth state to pick up the ban field
      const authData = queryClient.getQueryData<{ user?: { id: string } }>(['auth', 'me']);
      if (authData?.user?.id === event.data.user_id) {
        queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
      }
    });

    // Handle member unbanned
    connection.on('member.unbanned', (event) => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'members'] });
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'bans'] });

      // If the current user was unbanned, refresh auth state to clear the ban field
      const authData = queryClient.getQueryData<{ user?: { id: string } }>(['auth', 'me']);
      if (authData?.user?.id === event.data.user_id) {
        queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
        toast('You have been unbanned', 'success');
      }
    });

    // Handle typing events
    connection.on('typing.start', (event) => {
      addTypingUser(event.data.channel_id, event.data);
    });

    connection.on('typing.stop', (event) => {
      removeTypingUser(event.data.channel_id, event.data.user_id);
    });

    // Handle presence events
    connection.on('presence.changed', (event) => {
      setUserPresence(event.data.user_id, event.data.status);
    });

    // Handle initial presence (sent on connection with list of online users)
    connection.on('presence.initial', (event) => {
      const onlineUserIds = event.data.online_user_ids as string[];
      for (const userId of onlineUserIds) {
        setUserPresence(userId, 'online');
      }
    });

    // Handle notification events
    connection.on('notification', (event) => {
      const notification = event.data as NotificationData;

      // Increment notification_count for the channel
      if (notification.channel_id && notification.type !== 'thread_reply') {
        queryClient.setQueryData(
          ['channels', workspaceId],
          (old: { channels: ChannelWithMembership[] } | undefined) => {
            if (!old) return old;
            return {
              ...old,
              channels: old.channels.map((c) =>
                c.id === notification.channel_id
                  ? { ...c, notification_count: c.notification_count + 1 }
                  : c,
              ),
            };
          },
        );
        queryClient.invalidateQueries({ queryKey: ['workspaces', 'notifications'] });
      }

      // Only play sound and show browser notification when tab is not focused
      if (!document.hasFocus()) {
        playNotificationSound();

        const title = getNotificationTitle(notification);
        showBrowserNotification(title, notification.preview || '', () => {
          if (notification.channel_id && workspaceId) {
            navigateRef.current(`/workspaces/${workspaceId}/channels/${notification.channel_id}`);
          }
        });
      }
    });

    // Unlock audio on first interaction
    const handleInteraction = () => {
      unlockAudio();
      document.removeEventListener('click', handleInteraction);
      document.removeEventListener('keydown', handleInteraction);
    };
    document.addEventListener('click', handleInteraction);
    document.addEventListener('keydown', handleInteraction);

    connection.connect();

    // Request notification permission if not yet decided (avoids prompting on every connection)
    if ('Notification' in window && Notification.permission === 'default') {
      requestNotificationPermission();
    }

    return () => {
      connection.disconnect();
      connectionRef.current = null;
      setHasBeenConnected(false);
      setIsConnected(false);
    };
  }, [workspaceId, queryClient]);

  const isReconnecting = hasBeenConnected && !isConnected;

  return { isReconnecting };
}

// Helper to format notification title
function getNotificationTitle(notification: NotificationData): string {
  const prefix = notification.type === 'dm' ? 'DM' : `#${notification.channel_name || 'channel'}`;
  const sender = notification.sender_name || 'Someone';

  switch (notification.type) {
    case 'mention':
      return `${sender} mentioned you in ${prefix}`;
    case 'dm':
      return `${sender} sent you a message`;
    case 'channel':
      return `${sender} in ${prefix} (@channel)`;
    case 'here':
      return `${sender} in ${prefix} (@here)`;
    case 'everyone':
      return `${sender} in ${prefix} (@everyone)`;
    case 'thread_reply':
      return `${sender} replied to a thread in ${prefix}`;
    default:
      return `New message from ${sender}`;
  }
}
