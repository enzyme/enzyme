import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  workspacesApi,
  type CreateWorkspaceInput,
  type UpdateWorkspaceInput,
  type CreateInviteInput,
  type WorkspaceRole,
  type WorkspaceSummary,
  type WorkspaceNotificationSummary,
} from '@enzyme/api-client';
import { authKeys, workspaceKeys } from '../queryKeys';

export function useWorkspace(workspaceId: string | undefined) {
  return useQuery({
    queryKey: workspaceKeys.detail(workspaceId!),
    queryFn: () => workspacesApi.get(workspaceId!),
    enabled: !!workspaceId,
  });
}

export function useWorkspaceMembers(workspaceId: string | undefined) {
  return useQuery({
    queryKey: workspaceKeys.members(workspaceId!),
    queryFn: () => workspacesApi.listMembers(workspaceId!),
    enabled: !!workspaceId,
  });
}

export function useCreateWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateWorkspaceInput) => workspacesApi.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useUpdateWorkspace(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: UpdateWorkspaceInput) => workspacesApi.update(workspaceId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(workspaceId) });
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useUpdateMemberRole(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: WorkspaceRole }) =>
      workspacesApi.updateMemberRole(workspaceId, userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useRemoveMember(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => workspacesApi.removeMember(workspaceId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
    },
  });
}

export function useLeaveWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (workspaceId: string) => workspacesApi.leave(workspaceId),
    onSuccess: (_data, workspaceId) => {
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
      queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) });
    },
  });
}

export function useCreateInvite(workspaceId: string) {
  return useMutation({
    mutationFn: (input: CreateInviteInput) => workspacesApi.createInvite(workspaceId, input),
  });
}

export function useAcceptInvite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (code: string) => workspacesApi.acceptInvite(code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useUploadWorkspaceIcon(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (file: File) => workspacesApi.uploadIcon(workspaceId, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(workspaceId) });
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useDeleteWorkspaceIcon(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => workspacesApi.deleteIcon(workspaceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(workspaceId) });
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useWorkspaceNotifications() {
  const query = useQuery({
    queryKey: workspaceKeys.notifications(),
    queryFn: () => workspacesApi.getNotifications(),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
  });

  const notificationMap = useMemo(() => {
    const map = new Map<string, WorkspaceNotificationSummary>();
    if (query.data?.workspaces) {
      for (const ws of query.data.workspaces) {
        map.set(ws.workspace_id, ws);
      }
    }
    return map;
  }, [query.data]);

  return { ...query, notificationMap };
}

export function useReorderWorkspaces() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (workspaceIds: string[]) => workspacesApi.reorder(workspaceIds),
    onMutate: async (workspaceIds) => {
      await queryClient.cancelQueries({ queryKey: authKeys.me() });

      const previousData = queryClient.getQueryData<{
        user: unknown;
        workspaces: WorkspaceSummary[];
      }>(authKeys.me());

      // Optimistically reorder workspaces
      queryClient.setQueryData(
        authKeys.me(),
        (old: { user: unknown; workspaces: WorkspaceSummary[] } | undefined) => {
          if (!old) return old;

          // Create a map for quick lookup
          const workspaceMap = new Map(old.workspaces.map((ws) => [ws.id, ws]));

          // Reorder based on the new order
          const reordered: WorkspaceSummary[] = [];
          workspaceIds.forEach((id, index) => {
            const ws = workspaceMap.get(id);
            if (ws) {
              reordered.push({ ...ws, sort_order: index });
            }
          });

          // Add any workspaces not in the list (shouldn't happen but just in case)
          const inList = new Set(workspaceIds);
          old.workspaces.forEach((ws) => {
            if (!inList.has(ws.id)) {
              reordered.push(ws);
            }
          });

          return { ...old, workspaces: reordered };
        },
      );

      return { previousData };
    },
    onError: (_err, _workspaceIds, context) => {
      if (context?.previousData) {
        queryClient.setQueryData(authKeys.me(), context.previousData);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}
