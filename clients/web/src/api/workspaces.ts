import { get, post, type Workspace, type WorkspaceMemberWithUser, type Invite, type WorkspaceRole } from '@feather/api-client';

export interface CreateWorkspaceInput {
  slug: string;
  name: string;
}

export interface UpdateWorkspaceInput {
  slug?: string;
  name?: string;
}

export interface CreateInviteInput {
  invited_email?: string;
  role: WorkspaceRole;
  max_uses?: number;
  expires_in_hours?: number;
}

export const workspacesApi = {
  create: (input: CreateWorkspaceInput) =>
    post<{ workspace: Workspace }>('/workspaces/create', input),

  get: (workspaceId: string) =>
    get<{ workspace: Workspace }>(`/workspaces/${workspaceId}`),

  update: (workspaceId: string, input: UpdateWorkspaceInput) =>
    post<{ workspace: Workspace }>(`/workspaces/${workspaceId}/update`, input),

  listMembers: (workspaceId: string) =>
    post<{ members: WorkspaceMemberWithUser[] }>(`/workspaces/${workspaceId}/members/list`),

  removeMember: (workspaceId: string, userId: string) =>
    post<{ success: boolean }>(`/workspaces/${workspaceId}/members/remove`, { user_id: userId }),

  updateMemberRole: (workspaceId: string, userId: string, role: WorkspaceRole) =>
    post<{ success: boolean }>(`/workspaces/${workspaceId}/members/update-role`, { user_id: userId, role }),

  createInvite: (workspaceId: string, input: CreateInviteInput) =>
    post<{ invite: Invite }>(`/workspaces/${workspaceId}/invites/create`, input),

  acceptInvite: (code: string) =>
    post<{ workspace: Workspace }>(`/invites/${code}/accept`),

  // Typing endpoints
  startTyping: (workspaceId: string, channelId: string) =>
    post<{ success: boolean }>(`/workspaces/${workspaceId}/typing/start`, { channel_id: channelId }),

  stopTyping: (workspaceId: string, channelId: string) =>
    post<{ success: boolean }>(`/workspaces/${workspaceId}/typing/stop`, { channel_id: channelId }),
};
