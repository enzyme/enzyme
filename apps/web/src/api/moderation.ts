import { apiClient, throwIfError } from '@enzyme/api-client';

export const moderationApi = {
  // Pinning
  pinMessage: (messageId: string) =>
    throwIfError(apiClient.POST('/messages/{id}/pin', { params: { path: { id: messageId } } })),

  unpinMessage: (messageId: string) =>
    throwIfError(apiClient.POST('/messages/{id}/unpin', { params: { path: { id: messageId } } })),

  listPinnedMessages: (channelId: string, input?: { cursor?: string; limit?: number }) =>
    throwIfError(
      apiClient.POST('/channels/{id}/pins/list', {
        params: { path: { id: channelId } },
        body: input || {},
      }),
    ),

  // Banning
  banUser: (
    workspaceId: string,
    input: { user_id: string; reason?: string; duration_hours?: number; hide_messages: boolean },
  ) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/bans/create', {
        params: { path: { wid: workspaceId } },
        body: input,
      }),
    ),

  unbanUser: (workspaceId: string, userId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/bans/remove', {
        params: { path: { wid: workspaceId } },
        body: { user_id: userId },
      }),
    ),

  listBans: (workspaceId: string, input?: { cursor?: string; limit?: number }) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/bans/list', {
        params: { path: { wid: workspaceId } },
        body: input || {},
      }),
    ),

  // Blocking (workspace-scoped)
  blockUser: (workspaceId: string, userId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/blocks/create', {
        params: { path: { wid: workspaceId } },
        body: { user_id: userId },
      }),
    ),

  unblockUser: (workspaceId: string, userId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/blocks/remove', {
        params: { path: { wid: workspaceId } },
        body: { user_id: userId },
      }),
    ),

  listBlocks: (workspaceId: string) =>
    throwIfError(
      apiClient.GET('/workspaces/{wid}/blocks/list', {
        params: { path: { wid: workspaceId } },
      }),
    ),

  // Moderation log
  listModerationLog: (workspaceId: string, input?: { cursor?: string; limit?: number }) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/moderation-log/list', {
        params: { path: { wid: workspaceId } },
        body: input || {},
      }),
    ),
};
