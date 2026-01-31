import { post, type Channel, type ChannelWithMembership, type ChannelMember, type ChannelType } from '@feather/api-client';

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

export const channelsApi = {
  create: (workspaceId: string, input: CreateChannelInput) =>
    post<{ channel: Channel }>(`/workspaces/${workspaceId}/channels/create`, input),

  list: (workspaceId: string) =>
    post<{ channels: ChannelWithMembership[] }>(`/workspaces/${workspaceId}/channels/list`),

  createDM: (workspaceId: string, input: CreateDMInput) =>
    post<{ channel: Channel }>(`/workspaces/${workspaceId}/channels/dm`, input),

  update: (channelId: string, input: UpdateChannelInput) =>
    post<{ channel: Channel }>(`/channels/${channelId}/update`, input),

  archive: (channelId: string) =>
    post<{ success: boolean }>(`/channels/${channelId}/archive`),

  addMember: (channelId: string, userId: string, role?: string) =>
    post<{ success: boolean }>(`/channels/${channelId}/members/add`, { user_id: userId, role }),

  listMembers: (channelId: string) =>
    post<{ members: ChannelMember[] }>(`/channels/${channelId}/members/list`),

  join: (channelId: string) =>
    post<{ success: boolean }>(`/channels/${channelId}/join`),

  leave: (channelId: string) =>
    post<{ success: boolean }>(`/channels/${channelId}/leave`),
};
