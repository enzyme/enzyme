import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { channelKeys, unreadKeys } from '@enzyme/shared';
import { useSSE } from './useSSE';
import { useAppState } from './useAppState';

export function useSSELifecycle(workspaceId: string | null): {
  isReconnecting: boolean;
} {
  const queryClient = useQueryClient();
  const [isActive, setIsActive] = useState(true);

  // Pass undefined when backgrounded to disconnect SSE (triggers useSSE cleanup)
  const effectiveId = isActive ? (workspaceId ?? undefined) : undefined;
  const { isReconnecting } = useSSE(effectiveId);

  useAppState({
    onForeground: () => {
      setIsActive(true);
      // Invalidate workspace-scoped queries to refetch stale data after resume
      if (workspaceId) {
        queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) });
        queryClient.invalidateQueries({ queryKey: unreadKeys.list(workspaceId) });
      }
    },
    onBackground: () => {
      setIsActive(false);
    },
  });

  return { isReconnecting };
}
