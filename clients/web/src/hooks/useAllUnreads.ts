import { useInfiniteQuery } from '@tanstack/react-query';
import { post, type UnreadMessagesResult } from '@feather/api-client';

interface UseAllUnreadsOptions {
  workspaceId: string;
  enabled?: boolean;
}

export function useAllUnreads({ workspaceId, enabled = true }: UseAllUnreadsOptions) {
  return useInfiniteQuery<UnreadMessagesResult>({
    queryKey: ['unreads', workspaceId],
    queryFn: async ({ pageParam }) => {
      return post<UnreadMessagesResult>(`/workspaces/${workspaceId}/unreads`, {
        limit: 50,
        cursor: pageParam as string | undefined,
      });
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.next_cursor,
    enabled: enabled && !!workspaceId,
    staleTime: 30000, // 30 seconds
    refetchInterval: 60000, // Refetch every minute
  });
}
