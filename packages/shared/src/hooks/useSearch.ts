import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { messagesApi } from '@enzyme/api-client';
import { searchKeys } from '../queryKeys';

export interface UseSearchOptions {
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
    queryKey: searchKeys.query(workspaceId, query, channelId, userId, before, after, limit, offset),
    queryFn: () =>
      messagesApi.search(workspaceId, {
        query,
        channel_id: channelId,
        user_id: userId,
        before,
        after,
        limit,
        offset,
      }),
    enabled: !!workspaceId && query.trim().length > 0,
    placeholderData: keepPreviousData,
    staleTime: 30 * 1000,
  });
}
