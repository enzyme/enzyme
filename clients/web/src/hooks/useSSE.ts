import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { SSEConnection } from '../lib/sse';
import { usePresenceStore } from '../stores/presenceStore';
import type { MessageWithUser, Reaction, Channel, TypingEventData, PresenceData, MessageListResult, ChannelWithMembership, ChannelReadEventData } from '@feather/api-client';

export function useSSE(workspaceId: string | undefined) {
  const [isConnected, setIsConnected] = useState(false);
  const connectionRef = useRef<SSEConnection | null>(null);
  const queryClient = useQueryClient();
  const { addTypingUser, removeTypingUser, setUserPresence } = usePresenceStore();

  useEffect(() => {
    if (!workspaceId) {
      return;
    }

    const connection = new SSEConnection(workspaceId);
    connectionRef.current = connection;

    // Handle connected event
    connection.on('connected', () => {
      setIsConnected(true);
    });

    // Handle new message
    connection.on('message.new', (event) => {
      const message = event.data as MessageWithUser;

      // Add to channel messages
      queryClient.setQueryData(
        ['messages', message.channel_id],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;

          // Check if message already exists
          const exists = old.pages.some((page) =>
            page.messages.some((m) => m.id === message.id)
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
        }
      );

      // If it's a thread reply, update thread cache and parent reply count
      if (message.thread_parent_id) {
        queryClient.setQueryData(
          ['thread', message.thread_parent_id],
          (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
            if (!old) return old;

            // Check if message already exists in thread
            const exists = old.pages.some((page) =>
              page.messages.some((m) => m.id === message.id)
            );
            if (exists) return old;

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
          }
        );
      }

      // Increment unread count for channels (only for non-thread messages from other users)
      if (!message.thread_parent_id) {
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
                  c.id === message.channel_id
                    ? { ...c, unread_count: c.unread_count + 1 }
                    : c
                ),
              };
            }
          );
        }
      }
    });

    // Handle message updated
    connection.on('message.updated', (event) => {
      const message = event.data as MessageWithUser;

      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((m) =>
                m.id === message.id ? { ...m, ...message } : m
              ),
            })),
          };
        }
      );
    });

    // Handle message deleted
    connection.on('message.deleted', (event) => {
      const { id } = event.data as { id: string };

      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.filter((m) => m.id !== id),
            })),
          };
        }
      );
    });

    // Handle reaction added
    connection.on('reaction.added', (event) => {
      const reaction = event.data as Reaction;

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
                if (reactions.some((r) => r.user_id === reaction.user_id && r.emoji === reaction.emoji)) {
                  // Replace temp reaction with real one
                  return {
                    ...m,
                    reactions: reactions.map((r) =>
                      r.user_id === reaction.user_id && r.emoji === reaction.emoji ? reaction : r
                    ),
                  };
                }
                return { ...m, reactions: [...reactions, reaction] };
              }),
            })),
          };
        }
      );
    });

    // Handle reaction removed
    connection.on('reaction.removed', (event) => {
      const { message_id, user_id, emoji } = event.data as {
        message_id: string;
        user_id: string;
        emoji: string;
      };

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
                  reactions: reactions.filter(
                    (r) => !(r.user_id === user_id && r.emoji === emoji)
                  ),
                };
              }),
            })),
          };
        }
      );
    });

    // Handle channel events
    connection.on('channel.created', () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    });

    connection.on('channel.updated', (event) => {
      const channel = event.data as Channel;
      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: Channel[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) =>
              c.id === channel.id ? { ...c, ...channel } : c
            ),
          };
        }
      );
    });

    connection.on('channel.archived', () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    });

    connection.on('channel.member_added', () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    });

    connection.on('channel.member_removed', () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    });

    // Handle channel read events (for syncing across tabs/devices)
    connection.on('channel.read', (event) => {
      const data = event.data as ChannelReadEventData;
      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) =>
              c.id === data.channel_id
                ? { ...c, unread_count: 0, last_read_message_id: data.last_read_message_id }
                : c
            ),
          };
        }
      );
    });

    // Handle typing events
    connection.on('typing.start', (event) => {
      const data = event.data as TypingEventData;
      addTypingUser(data.channel_id, data);
    });

    connection.on('typing.stop', (event) => {
      const data = event.data as TypingEventData;
      removeTypingUser(data.channel_id, data.user_id);
    });

    // Handle presence events
    connection.on('presence.changed', (event) => {
      const data = event.data as PresenceData;
      setUserPresence(data.user_id, data.status);
    });

    connection.connect();

    return () => {
      connection.disconnect();
      connectionRef.current = null;
      setIsConnected(false);
    };
  }, [workspaceId, queryClient, addTypingUser, removeTypingUser, setUserPresence]);

  return { isConnected };
}
