import {
  get,
  post,
  del,
  type Channel,
  type ChannelWithMembership,
  type ChannelMember,
  type ChannelType,
  type MarkReadResponse,
  type NotificationPreferences,
  type DMSuggestionsResponse,
} from '@feather/api-client';

export interface CreateChannelInput {
  name: string;
  description?: string;
  type: ChannelType;
}

export interface UpdateChannelInput {
  name?: string;
  description?: string;
}

export interface CreateDMInput {
  user_ids: string[];
}

export interface ConvertGroupDMInput {
  name: string;
  description?: string;
  type?: 'public' | 'private';
}

export const channelsApi = {
  create: (workspaceId: string, input: CreateChannelInput) =>
    post<{ channel: Channel }>(`/workspaces/${workspaceId}/channels/create`, input),

  list: (workspaceId: string) =>
    post<{ channels: ChannelWithMembership[] }>(`/workspaces/${workspaceId}/channels/list`),

  createDM: (workspaceId: string, input: CreateDMInput) =>
    post<{ channel: Channel }>(`/workspaces/${workspaceId}/channels/dm`, input),

  update: (channelId: string, input: UpdateChannelInput) =>
    post<{ channel: Channel }>(`/channels/${channelId}/update`, input),

  archive: (channelId: string) => post<{ success: boolean }>(`/channels/${channelId}/archive`),

  addMember: (channelId: string, userId: string, role?: string) =>
    post<{ success: boolean }>(`/channels/${channelId}/members/add`, { user_id: userId, role }),

  listMembers: (channelId: string) =>
    post<{ members: ChannelMember[] }>(`/channels/${channelId}/members/list`),

  join: (channelId: string) => post<{ success: boolean }>(`/channels/${channelId}/join`),

  leave: (channelId: string) => post<{ success: boolean }>(`/channels/${channelId}/leave`),

  markAsRead: (channelId: string, messageId?: string) =>
    post<MarkReadResponse>(
      `/channels/${channelId}/mark-read`,
      messageId ? { message_id: messageId } : {},
    ),

  markAllAsRead: (workspaceId: string) =>
    post<{ success: boolean }>(`/workspaces/${workspaceId}/channels/mark-all-read`),

  getNotifications: (channelId: string) =>
    get<{ preferences: NotificationPreferences }>(`/channels/${channelId}/notifications`),

  updateNotifications: (channelId: string, preferences: NotificationPreferences) =>
    post<{ preferences: NotificationPreferences }>(
      `/channels/${channelId}/notifications`,
      preferences,
    ),

  star: (channelId: string) => post<{ success: boolean }>(`/channels/${channelId}/star`),

  unstar: (channelId: string) => del<{ success: boolean }>(`/channels/${channelId}/star`),

  getDMSuggestions: (workspaceId: string) =>
    post<DMSuggestionsResponse>(`/workspaces/${workspaceId}/dm-suggestions`),

  convertGroupDM: (channelId: string, input: ConvertGroupDMInput) =>
    post<{ channel: Channel }>(`/channels/${channelId}/convert`, input),
};
