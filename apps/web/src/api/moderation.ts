import {
  post,
  get,
  type MessageWithUser,
  type BanWithUser,
  type BlockWithUser,
  type ModerationLogEntryWithActor,
} from '@enzyme/api-client';

export const moderationApi = {
  // Pinning
  pinMessage: (messageId: string) =>
    post<{ message: MessageWithUser }>(`/messages/${messageId}/pin`),

  unpinMessage: (messageId: string) =>
    post<{ message: MessageWithUser }>(`/messages/${messageId}/unpin`),

  listPinnedMessages: (channelId: string, input?: { cursor?: string; limit?: number }) =>
    post<{ messages: MessageWithUser[]; has_more: boolean; next_cursor?: string }>(
      `/channels/${channelId}/pins/list`,
      input || {},
    ),

  // Banning
  banUser: (
    workspaceId: string,
    input: { user_id: string; reason?: string; duration_hours?: number; hide_messages?: boolean },
  ) => post<{ ban: BanWithUser }>(`/workspaces/${workspaceId}/bans/create`, input),

  unbanUser: (workspaceId: string, userId: string) =>
    post<{ success: boolean }>(`/workspaces/${workspaceId}/bans/remove`, { user_id: userId }),

  listBans: (workspaceId: string, input?: { cursor?: string; limit?: number }) =>
    post<{ bans: BanWithUser[]; has_more: boolean; next_cursor?: string }>(
      `/workspaces/${workspaceId}/bans/list`,
      input || {},
    ),

  // Blocking
  blockUser: (userId: string) => post<{ success: boolean }>('/users/blocks/create', { user_id: userId }),

  unblockUser: (userId: string) =>
    post<{ success: boolean }>('/users/blocks/remove', { user_id: userId }),

  listBlocks: () => get<{ blocks: BlockWithUser[] }>('/users/blocks/list'),

  // Moderation log
  listModerationLog: (workspaceId: string, input?: { cursor?: string; limit?: number }) =>
    post<{ entries: ModerationLogEntryWithActor[]; has_more: boolean; next_cursor?: string }>(
      `/workspaces/${workspaceId}/moderation-log/list`,
      input || {},
    ),
};
