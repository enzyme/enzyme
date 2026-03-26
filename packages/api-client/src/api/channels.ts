import { apiClient, throwIfError } from '../client';
import type {
  ChannelRole,
  CreateChannelInput,
  CreateDMInput,
  ConvertGroupDMInput,
  UpdateChannelInput,
  NotificationPreferences,
} from '../types';

export const channelsApi = {
  create: (workspaceId: string, input: CreateChannelInput) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/channels/create', {
        params: { path: { wid: workspaceId } },
        body: input,
      }),
    ),

  list: (workspaceId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/channels/list', {
        params: { path: { wid: workspaceId } },
      }),
    ),

  createDM: (workspaceId: string, input: CreateDMInput) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/channels/dm', {
        params: { path: { wid: workspaceId } },
        body: input,
      }),
    ),

  update: (channelId: string, input: UpdateChannelInput) =>
    throwIfError(
      apiClient.POST('/channels/{id}/update', {
        params: { path: { id: channelId } },
        body: input,
      }),
    ),

  archive: (channelId: string) =>
    throwIfError(apiClient.POST('/channels/{id}/archive', { params: { path: { id: channelId } } })),

  addMember: (channelId: string, userId: string, role?: ChannelRole) =>
    throwIfError(
      apiClient.POST('/channels/{id}/members/add', {
        params: { path: { id: channelId } },
        body: { user_id: userId, role },
      }),
    ),

  listMembers: (channelId: string) =>
    throwIfError(
      apiClient.POST('/channels/{id}/members/list', { params: { path: { id: channelId } } }),
    ),

  join: (channelId: string) =>
    throwIfError(apiClient.POST('/channels/{id}/join', { params: { path: { id: channelId } } })),

  leave: (channelId: string) =>
    throwIfError(apiClient.POST('/channels/{id}/leave', { params: { path: { id: channelId } } })),

  markAsRead: (channelId: string, messageId?: string) =>
    throwIfError(
      apiClient.POST('/channels/{id}/mark-read', {
        params: { path: { id: channelId } },
        body: messageId ? { message_id: messageId } : {},
      }),
    ),

  markAllAsRead: (workspaceId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/channels/mark-all-read', {
        params: { path: { wid: workspaceId } },
      }),
    ),

  getNotifications: (channelId: string) =>
    throwIfError(
      apiClient.GET('/channels/{id}/notifications', { params: { path: { id: channelId } } }),
    ),

  updateNotifications: (channelId: string, preferences: NotificationPreferences) =>
    throwIfError(
      apiClient.POST('/channels/{id}/notifications', {
        params: { path: { id: channelId } },
        body: preferences,
      }),
    ),

  star: (channelId: string) =>
    throwIfError(apiClient.POST('/channels/{id}/star', { params: { path: { id: channelId } } })),

  unstar: (channelId: string) =>
    throwIfError(apiClient.DELETE('/channels/{id}/star', { params: { path: { id: channelId } } })),

  convertGroupDM: (channelId: string, input: ConvertGroupDMInput) =>
    throwIfError(
      apiClient.POST('/channels/{id}/convert', {
        params: { path: { id: channelId } },
        body: input,
      }),
    ),
};
