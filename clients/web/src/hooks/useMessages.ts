import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { messagesApi, type SendMessageInput } from '../api/messages';
import type { MessageWithUser, MessageListResult } from '@feather/api-client';

const PAGE_SIZE = 50;

export function useMessages(channelId: string | undefined) {
  return useInfiniteQuery({
    queryKey: ['messages', channelId],
    queryFn: ({ pageParam }) =>
      messagesApi.list(channelId!, {
        cursor: pageParam,
        limit: PAGE_SIZE,
        direction: 'before',
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage: MessageListResult) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!channelId,
    staleTime: 1000 * 30, // 30 seconds
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
            page.messages.some((m) => m.id === data.message.id)
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
        }
      );
    },
  });
}

export function useSendThreadReply(parentMessageId: string, channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (content: string) =>
      messagesApi.send(channelId, { content, thread_parent_id: parentMessageId }),
    onSuccess: (data) => {
      // Add to thread cache (threads are ordered ASC, so append to end)
      queryClient.setQueryData(
        ['thread', parentMessageId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;

          // Check if message already exists (SSE might have added it first)
          const exists = old.pages.some((page) =>
            page.messages.some((m) => m.id === data.message.id)
          );
          if (exists) return old;

          const newPages = [...old.pages];
          const lastPageIndex = newPages.length - 1;
          if (newPages[lastPageIndex]) {
            newPages[lastPageIndex] = {
              ...newPages[lastPageIndex],
              messages: [...newPages[lastPageIndex].messages, data.message],
            };
          }
          return { ...old, pages: newPages };
        }
      );

      // Update reply count on parent message
      queryClient.setQueryData(
        ['messages', channelId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) =>
                msg.id === parentMessageId
                  ? { ...msg, reply_count: msg.reply_count + 1 }
                  : msg
              ),
            })),
          };
        }
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
              messages: page.messages.map((msg) =>
                msg.id === messageId ? data.message : msg
              ),
            })),
          };
        }
      );
    },
  });
}

export function useDeleteMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => messagesApi.delete(messageId),
    onSuccess: (_, messageId) => {
      // Remove from all message caches
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.filter((msg) => msg.id !== messageId),
            })),
          };
        }
      );
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
                    { id: 'temp', message_id: messageId, user_id: userId, emoji, created_at: new Date().toISOString() },
                  ],
                };
              }),
            })),
          };
        }
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
                  reactions: reactions.filter(
                    (r) => !(r.emoji === emoji && r.user_id === userId)
                  ),
                };
              }),
            })),
          };
        }
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
  updater: (messages: MessageWithUser[]) => MessageWithUser[]
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
    }
  );
}
