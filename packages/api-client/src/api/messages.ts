import { apiClient, throwIfError } from '../client';
import type { SendMessageInput, ListMessagesInput, SearchMessagesInput } from '../types';

export const messagesApi = {
  get: (messageId: string) =>
    throwIfError(apiClient.GET('/messages/{id}', { params: { path: { id: messageId } } })),

  send: (channelId: string, input: SendMessageInput) =>
    throwIfError(
      apiClient.POST('/channels/{id}/messages/send', {
        params: { path: { id: channelId } },
        body: input,
      }),
    ),

  list: (channelId: string, input?: ListMessagesInput) =>
    throwIfError(
      apiClient.POST('/channels/{id}/messages/list', {
        params: { path: { id: channelId } },
        body: input || {},
      }),
    ),

  update: (messageId: string, content: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/update', {
        params: { path: { id: messageId } },
        body: { content },
      }),
    ),

  delete: (messageId: string) =>
    throwIfError(apiClient.POST('/messages/{id}/delete', { params: { path: { id: messageId } } })),

  deleteLinkPreview: (messageId: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/link-preview/delete', {
        params: { path: { id: messageId } },
      }),
    ),

  addReaction: (messageId: string, emoji: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/reactions/add', {
        params: { path: { id: messageId } },
        body: { emoji },
      }),
    ),

  removeReaction: (messageId: string, emoji: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/reactions/remove', {
        params: { path: { id: messageId } },
        body: { emoji },
      }),
    ),

  listThread: (messageId: string, input?: ListMessagesInput) =>
    throwIfError(
      apiClient.POST('/messages/{id}/thread/list', {
        params: { path: { id: messageId } },
        body: input || {},
      }),
    ),

  markUnread: (messageId: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/mark-unread', { params: { path: { id: messageId } } }),
    ),

  getThreadSubscription: (messageId: string) =>
    throwIfError(
      apiClient.GET('/messages/{id}/subscription', { params: { path: { id: messageId } } }),
    ),

  subscribeToThread: (messageId: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/subscribe', { params: { path: { id: messageId } } }),
    ),

  unsubscribeFromThread: (messageId: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/unsubscribe', { params: { path: { id: messageId } } }),
    ),

  listUserThreads: (workspaceId: string, input?: { limit?: number; cursor?: string }) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/threads', {
        params: { path: { wid: workspaceId } },
        body: input || {},
      }),
    ),

  markThreadRead: (messageId: string, lastReadReplyId?: string) =>
    throwIfError(
      apiClient.POST('/messages/{id}/thread/mark-read', {
        params: { path: { id: messageId } },
        body: lastReadReplyId ? { last_read_reply_id: lastReadReplyId } : {},
      }),
    ),

  search: (workspaceId: string, input: SearchMessagesInput) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/messages/search', {
        params: { path: { wid: workspaceId } },
        body: input,
      }),
    ),
};
