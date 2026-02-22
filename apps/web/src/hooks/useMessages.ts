import { useInfiniteQuery, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { messagesApi, type SendMessageInput } from '../api/messages';
import type { MessageWithUser, MessageListResult } from '@enzyme/api-client';

const PAGE_SIZE = 50;

export function useMessage(messageId: string | undefined) {
  return useQuery({
    queryKey: ['message', messageId],
    queryFn: () => messagesApi.get(messageId!),
    enabled: !!messageId,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

interface PageParam {
  cursor?: string;
  direction: 'before' | 'after' | 'around';
}

export function useMessages(channelId: string | undefined, aroundMessageId?: string) {
  return useInfiniteQuery({
    queryKey: ['messages', channelId],
    queryFn: ({ pageParam }: { pageParam: PageParam }) =>
      messagesApi.list(channelId!, {
        cursor: pageParam.cursor,
        limit: PAGE_SIZE,
        direction: pageParam.direction === 'around' ? 'around' : pageParam.direction,
      }),
    initialPageParam: (aroundMessageId
      ? { cursor: aroundMessageId, direction: 'around' }
      : { direction: 'before' }) as PageParam,
    getNextPageParam: (lastPage: MessageListResult) =>
      lastPage.has_more
        ? { cursor: lastPage.next_cursor, direction: 'before' as const }
        : undefined,
    getPreviousPageParam: (
      firstPage: MessageListResult,
      _allPages: MessageListResult[],
      firstPageParam: PageParam,
    ) => {
      // "after" pages are ASC (oldest first), "before"/"around" are DESC (newest first)
      const newestId =
        firstPageParam.direction === 'after'
          ? firstPage.messages[firstPage.messages.length - 1]?.id
          : firstPage.messages[0]?.id;
      if (!newestId) return undefined;

      // "around" queries: API explicitly sets has_newer
      if (firstPage.has_newer) return { cursor: newestId, direction: 'after' as const };
      // "after" queries: has_more means more newer messages exist beyond this page
      if (firstPageParam.direction === 'after' && firstPage.has_more)
        return { cursor: newestId, direction: 'after' as const };
      // "before" queries with cursor: initial live-edge page was evicted by maxPages
      if (firstPageParam.cursor && firstPageParam.direction === 'before')
        return { cursor: newestId, direction: 'after' as const };
      return undefined;
    },
    enabled: !!channelId,
    staleTime: 1000 * 30, // 30 seconds
    maxPages: 20,
  });
}

export function useThreadMessages(parentMessageId: string | undefined) {
  return useInfiniteQuery({
    queryKey: ['thread', parentMessageId],
    queryFn: ({ pageParam }) =>
      messagesApi.listThread(parentMessageId!, {
        cursor: pageParam,
        limit: PAGE_SIZE,
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage: MessageListResult) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!parentMessageId,
  });
}

export function useSendMessage(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: SendMessageInput) => messagesApi.send(channelId, input),
    onSuccess: (data) => {
      // Add message to cache (SSE may have already added it)
      queryClient.setQueryData(
        ['messages', channelId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;

          // Check if message already exists (SSE might have added it first)
          const exists = old.pages.some((page) =>
            page.messages.some((m) => m.id === data.message.id),
          );
          if (exists) return old;

          const newPages = [...old.pages];
          if (newPages[0]) {
            newPages[0] = {
              ...newPages[0],
              messages: [data.message, ...newPages[0].messages],
            };
          }
          return { ...old, pages: newPages };
        },
      );
    },
  });
}

interface SendThreadReplyInput {
  content?: string;
  attachment_ids?: string[];
  also_send_to_channel?: boolean;
}

export function useSendThreadReply(parentMessageId: string, channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: SendThreadReplyInput | string) => {
      const params = typeof input === 'string' ? { content: input } : input;
      return messagesApi.send(channelId, { ...params, thread_parent_id: parentMessageId });
    },
    onSuccess: (data) => {
      // Invalidate thread subscription - user may have been auto-subscribed
      queryClient.invalidateQueries({ queryKey: ['thread-subscription', parentMessageId] });

      // Check if message already exists in thread cache (SSE might have added it first)
      const threadData = queryClient.getQueryData<{
        pages: MessageListResult[];
        pageParams: (string | undefined)[];
      }>(['thread', parentMessageId]);
      const alreadyInCache =
        threadData?.pages.some((page) => page.messages.some((m) => m.id === data.message.id)) ??
        false;

      // Add to thread cache if not already there (threads are ordered ASC, so append to end)
      if (!alreadyInCache) {
        queryClient.setQueryData(
          ['thread', parentMessageId],
          (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
            if (!old) return old;

            const newPages = [...old.pages];
            const lastPageIndex = newPages.length - 1;
            if (newPages[lastPageIndex]) {
              newPages[lastPageIndex] = {
                ...newPages[lastPageIndex],
                messages: [...newPages[lastPageIndex].messages, data.message],
              };
            }
            return { ...old, pages: newPages };
          },
        );
      }

      // If also_send_to_channel, add to the channel message cache too
      if (data.message.also_send_to_channel) {
        queryClient.setQueryData(
          ['messages', channelId],
          (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
            if (!old) return old;
            const exists = old.pages.some((page) =>
              page.messages.some((m) => m.id === data.message.id),
            );
            if (exists) return old;
            const newPages = [...old.pages];
            if (newPages[0]) {
              newPages[0] = {
                ...newPages[0],
                messages: [data.message, ...newPages[0].messages],
              };
            }
            return { ...old, pages: newPages };
          },
        );
      }

      // Update parent message: reply_count (only if we added the message), last_reply_at, and thread_participants
      queryClient.setQueryData(
        ['messages', channelId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== parentMessageId) return msg;

                // Check if user should be added to thread_participants
                const participants = msg.thread_participants || [];
                const shouldAddParticipant =
                  data.message.user_id &&
                  !participants.some((p) => p.user_id === data.message.user_id);

                return {
                  ...msg,
                  // Only increment if SSE hasn't already done it
                  reply_count: alreadyInCache ? msg.reply_count : (msg.reply_count || 0) + 1,
                  last_reply_at: data.message.created_at,
                  thread_participants: shouldAddParticipant
                    ? [
                        ...participants,
                        {
                          user_id: data.message.user_id!,
                          display_name: data.message.user_display_name,
                          avatar_url: data.message.user_avatar_url,
                        },
                      ]
                    : participants,
                };
              }),
            })),
          };
        },
      );
    },
  });
}

export function useUpdateMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ messageId, content }: { messageId: string; content: string }) =>
      messagesApi.update(messageId, content),
    onSuccess: (data, { messageId }) => {
      // Update in all message caches
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => (msg.id === messageId ? data.message : msg)),
            })),
          };
        },
      );
    },
  });
}

export function useDeleteLinkPreview() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => messagesApi.deleteLinkPreview(messageId),
    onMutate: async (messageId) => {
      await queryClient.cancelQueries({ queryKey: ['messages'] });
      const previousData = queryClient.getQueriesData({ queryKey: ['messages'] });

      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          let changed = false;
          const pages = old.pages.map((page) => {
            if (!page.messages.some((m) => m.id === messageId && m.link_preview)) return page;
            changed = true;
            return {
              ...page,
              messages: page.messages.map((m) =>
                m.id === messageId ? { ...m, link_preview: undefined } : m,
              ),
            };
          });
          return changed ? { ...old, pages } : old;
        },
      );

      return { previousData };
    },
    onError: (_err, _messageId, context) => {
      if (context?.previousData) {
        for (const [key, data] of context.previousData) {
          queryClient.setQueryData(key, data);
        }
      }
    },
  });
}

export function useDeleteMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => messagesApi.delete(messageId),
    onSuccess: (_, messageId) => {
      // Find the message's thread_parent_id before removing it from caches
      let threadParentId: string | undefined;
      const threadQueries = queryClient.getQueriesData<{ pages: MessageListResult[] }>({
        queryKey: ['thread'],
      });
      for (const [, data] of threadQueries) {
        if (!data) continue;
        for (const page of data.pages) {
          const found = page.messages.find((m) => m.id === messageId);
          if (found?.thread_parent_id) {
            threadParentId = found.thread_parent_id;
            break;
          }
        }
        if (threadParentId) break;
      }

      // Update message caches: set deleted_at for messages with replies, filter out others
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages
                .map((msg) => {
                  if (msg.id !== messageId) return msg;
                  if (!threadParentId) threadParentId = msg.thread_parent_id;
                  // Messages with replies: mark as deleted (keep in cache for placeholder)
                  if (msg.reply_count > 0) {
                    return { ...msg, deleted_at: new Date().toISOString() };
                  }
                  // Messages without replies: return null to filter out
                  return null;
                })
                .filter((msg): msg is MessageWithUser => msg !== null),
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
              messages: page.messages.filter((msg) => msg.id !== messageId),
            })),
          };
        },
      );

      // Decrement parent's reply_count if this was a thread reply
      if (threadParentId) {
        queryClient.setQueriesData(
          { queryKey: ['messages'] },
          (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
            if (!old) return old;
            return {
              ...old,
              pages: old.pages.map((page) => ({
                ...page,
                messages: page.messages.map((msg) => {
                  if (msg.id !== threadParentId) return msg;
                  return { ...msg, reply_count: Math.max((msg.reply_count || 0) - 1, 0) };
                }),
              })),
            };
          },
        );
      }
    },
  });
}

export function useAddReaction(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ messageId, emoji }: { messageId: string; emoji: string }) =>
      messagesApi.addReaction(messageId, emoji),
    onMutate: async ({ messageId, emoji }) => {
      // Optimistic update
      await queryClient.cancelQueries({ queryKey: ['messages', channelId] });

      const previousData = queryClient.getQueryData(['messages', channelId]);
      const authData = queryClient.getQueryData<{ user?: { id: string } }>(['auth', 'me']);
      const userId = authData?.user?.id || 'temp';

      queryClient.setQueryData(
        ['messages', channelId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== messageId) return msg;
                const reactions = msg.reactions || [];
                // Check if reaction already exists (by user + emoji)
                if (reactions.some((r) => r.user_id === userId && r.emoji === emoji)) {
                  return msg;
                }
                return {
                  ...msg,
                  reactions: [
                    ...reactions,
                    {
                      id: 'temp',
                      message_id: messageId,
                      user_id: userId,
                      emoji,
                      created_at: new Date().toISOString(),
                    },
                  ],
                };
              }),
            })),
          };
        },
      );

      return { previousData };
    },
    onError: (_, __, context) => {
      if (context?.previousData) {
        queryClient.setQueryData(['messages', channelId], context.previousData);
      }
    },
  });
}

export function useRemoveReaction(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ messageId, emoji }: { messageId: string; emoji: string }) =>
      messagesApi.removeReaction(messageId, emoji),
    onMutate: async ({ messageId, emoji }) => {
      await queryClient.cancelQueries({ queryKey: ['messages', channelId] });

      const previousData = queryClient.getQueryData(['messages', channelId]);
      const authData = queryClient.getQueryData<{ user?: { id: string } }>(['auth', 'me']);
      const userId = authData?.user?.id;

      queryClient.setQueryData(
        ['messages', channelId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== messageId) return msg;
                const reactions = msg.reactions || [];
                return {
                  ...msg,
                  reactions: reactions.filter((r) => !(r.emoji === emoji && r.user_id === userId)),
                };
              }),
            })),
          };
        },
      );

      return { previousData };
    },
    onError: (_, __, context) => {
      if (context?.previousData) {
        queryClient.setQueryData(['messages', channelId], context.previousData);
      }
    },
  });
}

// Helper to update message cache from SSE events
export function updateMessageInCache(
  queryClient: ReturnType<typeof useQueryClient>,
  channelId: string,
  updater: (messages: MessageWithUser[]) => MessageWithUser[],
) {
  queryClient.setQueryData(
    ['messages', channelId],
    (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
      if (!old) return old;
      return {
        ...old,
        pages: old.pages.map((page) => ({
          ...page,
          messages: updater(page.messages),
        })),
      };
    },
  );
}

export function useMarkMessageUnread(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => messagesApi.markUnread(messageId),
    onSuccess: () => {
      // Invalidate channels to refresh unread counts
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}
