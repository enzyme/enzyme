import { useState, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useMessages } from './useMessages';
import { messagesApi, type MessageListResult } from '@enzyme/api-client';
import { messageKeys } from '@enzyme/shared';

const PAGE_SIZE = 50;

export function useVirtualMessages(channelId: string | undefined) {
  const [aroundMessageId, setAroundMessageId] = useState<string | undefined>(undefined);
  const [isDetached, setIsDetached] = useState(false);
  const queryClient = useQueryClient();

  const messagesQuery = useMessages(channelId, aroundMessageId);

  const jumpToMessage = useCallback(
    async (messageId: string) => {
      if (!channelId) return;

      // Check if message is already in cache
      const existing = queryClient.getQueryData<{
        pages: MessageListResult[];
        pageParams: unknown[];
      }>(messageKeys.list(channelId));

      if (existing) {
        const found = existing.pages.some((page) => page.messages.some((m) => m.id === messageId));
        if (found) {
          return { alreadyLoaded: true };
        }
      }

      // Fetch centered on the target message
      const result = await messagesApi.list(channelId, {
        cursor: messageId,
        limit: PAGE_SIZE,
        direction: 'around',
      });

      // Replace the query cache with the around-fetched data
      queryClient.setQueryData(messageKeys.list(channelId), {
        pages: [result],
        pageParams: [{ cursor: messageId, direction: 'around' }],
      });

      setIsDetached(true);
      return { alreadyLoaded: false };
    },
    [channelId, queryClient],
  );

  const jumpToLatest = useCallback(() => {
    if (!channelId) return;
    setAroundMessageId(undefined);
    setIsDetached(false);
    queryClient.resetQueries({ queryKey: messageKeys.list(channelId) });
  }, [channelId, queryClient]);

  const onReachBottom = useCallback(() => {
    // Exit detached mode when user scrolls to bottom and all newer pages are loaded
    if (isDetached && !messagesQuery.hasPreviousPage) {
      setIsDetached(false);
    }
  }, [isDetached, messagesQuery.hasPreviousPage]);

  return {
    ...messagesQuery,
    isDetached,
    jumpToMessage,
    jumpToLatest,
    onReachBottom,
  };
}
