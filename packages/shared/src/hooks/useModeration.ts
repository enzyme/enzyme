import {
  useQuery,
  useInfiniteQuery,
  useMutation,
  useQueryClient,
  QueryClient,
} from '@tanstack/react-query';
import { moderationApi, type Message, type MessageListResult } from '@enzyme/api-client';
import { messageKeys, pinnedMessageKeys, workspaceKeys } from '../queryKeys';

type MessagePages = { pages: MessageListResult[]; pageParams: (string | undefined)[] };

/** Shared helper to update a single message inside all cached message pages. */
export function updateModerationMessageInCache(
  queryClient: QueryClient,
  messageId: string,
  updater: (message: Message) => Message,
) {
  queryClient.setQueriesData({ queryKey: messageKeys.all }, (old: MessagePages | undefined) => {
    if (!old) return old;
    let changed = false;
    const pages = old.pages.map((page) => {
      if (!page.messages.some((m) => m.id === messageId)) return page;
      changed = true;
      return {
        ...page,
        messages: page.messages.map((m) => (m.id === messageId ? updater(m) : m)),
      };
    });
    return changed ? { ...old, pages } : old;
  });
}

// --- Pinning ---

export function usePinnedMessages(channelId: string | undefined) {
  return useQuery({
    queryKey: pinnedMessageKeys.list(channelId!),
    queryFn: () => moderationApi.listPinnedMessages(channelId!, { limit: 50 }),
    enabled: !!channelId,
    staleTime: 5 * 60 * 1000,
  });
}

export function usePinMessage(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => moderationApi.pinMessage(messageId),
    onSuccess: (data) => {
      updateModerationMessageInCache(queryClient, data.message.id, (m) => ({
        ...m,
        ...data.message,
      }));
      queryClient.invalidateQueries({ queryKey: pinnedMessageKeys.list(channelId) });
    },
  });
}

export function useUnpinMessage(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => moderationApi.unpinMessage(messageId),
    onSuccess: (data) => {
      updateModerationMessageInCache(queryClient, data.message.id, (m) => ({
        ...m,
        ...data.message,
      }));
      queryClient.invalidateQueries({ queryKey: pinnedMessageKeys.list(channelId) });
    },
  });
}

// --- Banning ---

export function useBans(workspaceId: string | undefined) {
  return useQuery({
    queryKey: workspaceKeys.bans(workspaceId!),
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
      hide_messages: boolean;
    }) => moderationApi.banUser(workspaceId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.bans(workspaceId) });
      queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
    },
  });
}

export function useUnbanUser(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => moderationApi.unbanUser(workspaceId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.bans(workspaceId) });
    },
  });
}

// --- Blocking (workspace-scoped) ---

export function useBlocks(workspaceId: string | undefined) {
  return useQuery({
    queryKey: workspaceKeys.blocks(workspaceId!),
    queryFn: () => moderationApi.listBlocks(workspaceId!),
    enabled: !!workspaceId,
  });
}

export function useBlockUser(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => moderationApi.blockUser(workspaceId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.blocks(workspaceId) });
      queryClient.invalidateQueries({ queryKey: messageKeys.all });
    },
  });
}

export function useUnblockUser(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => moderationApi.unblockUser(workspaceId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.blocks(workspaceId) });
      queryClient.invalidateQueries({ queryKey: messageKeys.all });
    },
  });
}

// --- Moderation Log ---

export function useModerationLog(workspaceId: string | undefined) {
  return useInfiniteQuery({
    queryKey: workspaceKeys.moderationLog(workspaceId!),
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
