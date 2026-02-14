import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { messagesApi } from '../api/messages';

interface UseSearchOptions {
  workspaceId: string;
  query: string;
  channelId?: string;
  userId?: string;
  before?: string;
  after?: string;
  limit?: number;
  offset?: number;
}

export function useSearch({
  workspaceId,
  query,
  channelId,
  userId,
  before,
  after,
  limit = 20,
  offset = 0,
}: UseSearchOptions) {
  return useQuery({
    queryKey: ['search', workspaceId, query, channelId, userId, before, after, limit, offset],
    queryFn: () =>
      messagesApi.search(workspaceId, {
        query,
        channel_id: channelId,
        user_id: userId,
        before: before ? before : undefined,
        after: after ? after : undefined,
        limit,
        offset,
      }),
    enabled: !!workspaceId && query.trim().length > 0,
    placeholderData: keepPreviousData,
    staleTime: 30 * 1000,
  });
}
