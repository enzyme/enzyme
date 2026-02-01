import { get, post, type MessageWithUser, type MessageListResult, type Reaction, type ThreadSubscriptionStatus } from '@feather/api-client';

export interface SendMessageInput {
  content?: string;
  thread_parent_id?: string;
  attachment_ids?: string[];
}

export interface ListMessagesInput {
  cursor?: string;
  limit?: number;
  direction?: 'before' | 'after';
}

export const messagesApi = {
  get: (messageId: string) =>
    get<{ message: MessageWithUser }>(`/messages/${messageId}`),

  send: (channelId: string, input: SendMessageInput) =>
    post<{ message: MessageWithUser }>(`/channels/${channelId}/messages/send`, input),

  list: (channelId: string, input?: ListMessagesInput) =>
    post<MessageListResult>(`/channels/${channelId}/messages/list`, input || {}),

  update: (messageId: string, content: string) =>
    post<{ message: MessageWithUser }>(`/messages/${messageId}/update`, { content }),

  delete: (messageId: string) =>
    post<{ success: boolean }>(`/messages/${messageId}/delete`),

  addReaction: (messageId: string, emoji: string) =>
    post<{ reaction: Reaction }>(`/messages/${messageId}/reactions/add`, { emoji }),

  removeReaction: (messageId: string, emoji: string) =>
    post<{ success: boolean }>(`/messages/${messageId}/reactions/remove`, { emoji }),

  listThread: (messageId: string, input?: ListMessagesInput) =>
    post<MessageListResult>(`/messages/${messageId}/thread/list`, input || {}),

  markUnread: (messageId: string) =>
    post<{ success: boolean }>(`/messages/${messageId}/mark-unread`),

  getThreadSubscription: (messageId: string) =>
    get<{ status: ThreadSubscriptionStatus }>(`/messages/${messageId}/subscription`),

  subscribeToThread: (messageId: string) =>
    post<{ status: ThreadSubscriptionStatus }>(`/messages/${messageId}/subscribe`),

  unsubscribeFromThread: (messageId: string) =>
    post<{ status: ThreadSubscriptionStatus }>(`/messages/${messageId}/unsubscribe`),
};
