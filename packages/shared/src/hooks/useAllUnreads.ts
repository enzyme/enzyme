import { useInfiniteQuery } from '@tanstack/react-query';
import { apiClient, throwIfError } from '@enzyme/api-client';
import type { UnreadMessagesResult } from '@enzyme/api-client';
import { unreadKeys } from '../queryKeys';

interface UseAllUnreadsOptions {
  workspaceId: string;
  enabled?: boolean;
}

export function useAllUnreads({ workspaceId, enabled = true }: UseAllUnreadsOptions) {
  return useInfiniteQuery<UnreadMessagesResult>({
    queryKey: unreadKeys.list(workspaceId),
    queryFn: async ({ pageParam }) => {
      return throwIfError(
        apiClient.POST('/workspaces/{wid}/unreads', {
          params: { path: { wid: workspaceId } },
          body: {
            limit: 50,
            cursor: pageParam as string | undefined,
          },
        }),
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.next_cursor,
    enabled: enabled && !!workspaceId,
    staleTime: 30000, // 30 seconds
    refetchInterval: 60000, // Refetch every minute
  });
}
