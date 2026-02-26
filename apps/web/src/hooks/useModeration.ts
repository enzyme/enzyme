import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { moderationApi } from '../api/moderation';
import type { MessageListResult } from '@enzyme/api-client';

// --- Pinning ---

export function usePinnedMessages(channelId: string | undefined) {
  return useQuery({
    queryKey: ['pinned-messages', channelId],
    queryFn: () => moderationApi.listPinnedMessages(channelId!, { limit: 50 }),
    enabled: !!channelId,
  });
}

export function usePinMessage(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => moderationApi.pinMessage(messageId),
    onSuccess: (data) => {
      // Update the message in cache with pinned_at/pinned_by
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          let changed = false;
          const pages = old.pages.map((page) => {
            if (!page.messages.some((m) => m.id === data.message.id)) return page;
            changed = true;
            return {
              ...page,
              messages: page.messages.map((m) =>
                m.id === data.message.id ? { ...m, ...data.message } : m,
              ),
            };
          });
          return changed ? { ...old, pages } : old;
        },
      );
      // Invalidate pinned messages list
      queryClient.invalidateQueries({ queryKey: ['pinned-messages', channelId] });
    },
  });
}

export function useUnpinMessage(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => moderationApi.unpinMessage(messageId),
    onSuccess: (data) => {
      // Update the message in cache to clear pinned_at/pinned_by
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          let changed = false;
          const pages = old.pages.map((page) => {
            if (!page.messages.some((m) => m.id === data.message.id)) return page;
            changed = true;
            return {
              ...page,
              messages: page.messages.map((m) =>
                m.id === data.message.id ? { ...m, ...data.message } : m,
              ),
            };
          });
          return changed ? { ...old, pages } : old;
        },
      );
      queryClient.invalidateQueries({ queryKey: ['pinned-messages', channelId] });
    },
  });
}

// --- Banning ---

export function useBans(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['workspace', workspaceId, 'bans'],
    queryFn: () => moderationApi.listBans(workspaceId!, { limit: 50 }),
    enabled: !!workspaceId,
  });
}

export function useBanUser(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: {
      user_id: string;
      reason?: string;
      duration_hours?: number;
      hide_messages?: boolean;
    }) => moderationApi.banUser(workspaceId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'bans'] });
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'members'] });
    },
  });
}

export function useUnbanUser(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => moderationApi.unbanUser(workspaceId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'bans'] });
    },
  });
}

// --- Blocking (workspace-scoped) ---

export function useBlocks(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['workspace', workspaceId, 'blocks'],
    queryFn: () => moderationApi.listBlocks(workspaceId!),
    enabled: !!workspaceId,
  });
}

export function useBlockUser(workspaceId: string | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => moderationApi.blockUser(workspaceId!, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'blocks'] });
    },
  });
}

export function useUnblockUser(workspaceId: string | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => moderationApi.unblockUser(workspaceId!, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'blocks'] });
    },
  });
}

// --- Moderation Log ---

export function useModerationLog(workspaceId: string | undefined) {
  return useInfiniteQuery({
    queryKey: ['workspace', workspaceId, 'moderation-log'],
    queryFn: ({ pageParam }) =>
      moderationApi.listModerationLog(workspaceId!, {
        cursor: pageParam,
        limit: 50,
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.next_cursor : undefined),
    enabled: !!workspaceId,
  });
}
