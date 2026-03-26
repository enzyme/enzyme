import { apiClient, throwIfError } from '../client';
import type { ScheduleMessageInput, UpdateScheduledMessageInput } from '../types';

export const scheduledMessagesApi = {
  schedule: (channelId: string, input: ScheduleMessageInput) =>
    throwIfError(
      apiClient.POST('/channels/{id}/messages/schedule', {
        params: { path: { id: channelId } },
        body: input,
      }),
    ),

  list: (workspaceId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/scheduled-messages', {
        params: { path: { wid: workspaceId } },
      }),
    ),

  get: (id: string) =>
    throwIfError(apiClient.POST('/scheduled-messages/{id}', { params: { path: { id } } })),

  update: (id: string, input: UpdateScheduledMessageInput) =>
    throwIfError(
      apiClient.POST('/scheduled-messages/{id}/update', { params: { path: { id } }, body: input }),
    ),

  delete: (id: string) =>
    throwIfError(apiClient.POST('/scheduled-messages/{id}/delete', { params: { path: { id } } })),

  sendNow: (id: string) =>
    throwIfError(apiClient.POST('/scheduled-messages/{id}/send-now', { params: { path: { id } } })),
};
