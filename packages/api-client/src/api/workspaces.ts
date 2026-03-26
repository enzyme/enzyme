import { apiClient, throwIfError, multipartRequest } from '../client';
import type {
  CreateWorkspaceInput,
  UpdateWorkspaceInput,
  CreateInviteInput,
  WorkspaceRole,
} from '../types';

export const workspacesApi = {
  create: (input: CreateWorkspaceInput) =>
    throwIfError(apiClient.POST('/workspaces/create', { body: input })),

  get: (workspaceId: string) =>
    throwIfError(apiClient.GET('/workspaces/{wid}', { params: { path: { wid: workspaceId } } })),

  update: (workspaceId: string, input: UpdateWorkspaceInput) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/update', {
        params: { path: { wid: workspaceId } },
        body: input,
      }),
    ),

  listMembers: (workspaceId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/members/list', {
        params: { path: { wid: workspaceId } },
      }),
    ),

  removeMember: (workspaceId: string, userId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/members/remove', {
        params: { path: { wid: workspaceId } },
        body: { user_id: userId },
      }),
    ),

  updateMemberRole: (workspaceId: string, userId: string, role: WorkspaceRole) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/members/update-role', {
        params: { path: { wid: workspaceId } },
        body: { user_id: userId, role },
      }),
    ),

  createInvite: (workspaceId: string, input: CreateInviteInput) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/invites/create', {
        params: { path: { wid: workspaceId } },
        body: input,
      }),
    ),

  acceptInvite: (code: string) =>
    throwIfError(apiClient.POST('/invites/{code}/accept', { params: { path: { code } } })),

  startTyping: (workspaceId: string, channelId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/typing/start', {
        params: { path: { wid: workspaceId } },
        body: { channel_id: channelId },
      }),
    ),

  stopTyping: (workspaceId: string, channelId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/typing/stop', {
        params: { path: { wid: workspaceId } },
        body: { channel_id: channelId },
      }),
    ),

  uploadIcon: (workspaceId: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return throwIfError(
      apiClient.POST('/workspaces/{wid}/icon', {
        params: { path: { wid: workspaceId } },
        ...multipartRequest(formData),
      }),
    );
  },

  deleteIcon: (workspaceId: string) =>
    throwIfError(
      apiClient.DELETE('/workspaces/{wid}/icon', { params: { path: { wid: workspaceId } } }),
    ),

  reorder: (workspaceIds: string[]) =>
    throwIfError(apiClient.POST('/workspaces/reorder', { body: { workspace_ids: workspaceIds } })),

  leave: (workspaceId: string) =>
    throwIfError(
      apiClient.POST('/workspaces/{wid}/leave', { params: { path: { wid: workspaceId } } }),
    ),

  getNotifications: () => throwIfError(apiClient.GET('/workspaces/notifications')),
};
