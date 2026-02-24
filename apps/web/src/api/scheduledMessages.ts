import {
  post,
  type ScheduledMessage,
  type ScheduleMessageInput,
  type UpdateScheduledMessageInput,
  type MessageWithUser,
} from '@enzyme/api-client';

export const scheduledMessagesApi = {
  schedule: (channelId: string, input: ScheduleMessageInput) =>
    post<{ scheduled_message: ScheduledMessage }>(`/channels/${channelId}/messages/schedule`, input),

  list: (workspaceId: string) =>
    post<{ scheduled_messages: ScheduledMessage[]; count: number }>(
      `/workspaces/${workspaceId}/scheduled-messages`,
    ),

  get: (id: string) =>
    post<{ scheduled_message: ScheduledMessage }>(`/scheduled-messages/${id}`),

  update: (id: string, input: UpdateScheduledMessageInput) =>
    post<{ scheduled_message: ScheduledMessage }>(`/scheduled-messages/${id}/update`, input),

  delete: (id: string) => post<{ success: boolean }>(`/scheduled-messages/${id}/delete`),

  sendNow: (id: string) =>
    post<{ message: MessageWithUser }>(`/scheduled-messages/${id}/send-now`),
};
