/**
 * Centralized query key factories for TanStack Query.
 *
 * Pattern: each domain exposes an object with key factories so that
 * invalidation, cancellation, and cache reads are always consistent.
 */

export const authKeys = {
  all: ['auth'] as const,
  me: () => ['auth', 'me'] as const,
};

export const userKeys = {
  all: ['user'] as const,
  detail: (userId: string) => ['user', userId] as const,
};

export const messageKeys = {
  all: ['messages'] as const,
  list: (channelId: string) => ['messages', channelId] as const,
  // Intentionally 'message' (singular) — single-message fetches are separate from
  // channel message lists and should not be bulk-invalidated by messageKeys.all.
  detail: (messageId: string) => ['message', messageId] as const,
};

export const threadKeys = {
  all: ['thread'] as const,
  detail: (parentMessageId: string) => ['thread', parentMessageId] as const,
  subscription: (messageId: string) => ['thread-subscription', messageId] as const,
  userThreads: (workspaceId: string) => ['user-threads', workspaceId] as const,
};

export const channelKeys = {
  all: ['channels'] as const,
  list: (workspaceId: string) => ['channels', workspaceId] as const,
  members: (channelId: string) => ['channel', channelId, 'members'] as const,
  notifications: (channelId: string) => ['channel-notifications', channelId] as const,
};

export const workspaceKeys = {
  all: ['workspace'] as const,
  detail: (workspaceId: string) => ['workspace', workspaceId] as const,
  members: (workspaceId: string) => ['workspace', workspaceId, 'members'] as const,
  bans: (workspaceId: string) => ['workspace', workspaceId, 'bans'] as const,
  blocks: (workspaceId: string) => ['workspace', workspaceId, 'blocks'] as const,
  moderationLog: (workspaceId: string) => ['workspace', workspaceId, 'moderation-log'] as const,
  // Intentionally outside 'workspace' prefix — this is a cross-workspace aggregate
  // that should not be invalidated by single-workspace operations.
  notifications: () => ['workspaces', 'notifications'] as const,
};

export const emojiKeys = {
  all: ['custom-emojis'] as const,
  list: (workspaceId: string) => ['custom-emojis', workspaceId] as const,
};

export const unreadKeys = {
  all: ['unreads'] as const,
  list: (workspaceId: string) => ['unreads', workspaceId] as const,
};

export const pinnedMessageKeys = {
  all: ['pinned-messages'] as const,
  list: (channelId: string) => ['pinned-messages', channelId] as const,
};

export const scheduledMessageKeys = {
  all: ['scheduled-messages'] as const,
  list: (workspaceId: string) => ['scheduled-messages', workspaceId] as const,
};

export const searchKeys = {
  all: ['search'] as const,
  query: (
    workspaceId: string,
    query: string,
    channelId?: string,
    userId?: string,
    before?: string,
    after?: string,
    limit?: number,
    offset?: number,
  ) => ['search', workspaceId, query, channelId, userId, before, after, limit, offset] as const,
};

export const serverKeys = {
  info: () => ['server-info'] as const,
};
