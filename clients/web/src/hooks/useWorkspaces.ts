import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  workspacesApi,
  type CreateWorkspaceInput,
  type CreateInviteInput,
} from '../api/workspaces';
import type {
  WorkspaceRole,
  WorkspaceSummary,
  WorkspaceNotificationSummary,
} from '@feather/api-client';

export function useWorkspace(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['workspace', workspaceId],
    queryFn: () => workspacesApi.get(workspaceId!),
    enabled: !!workspaceId,
  });
}

export function useWorkspaceMembers(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['workspace', workspaceId, 'members'],
    queryFn: () => workspacesApi.listMembers(workspaceId!),
    enabled: !!workspaceId,
  });
}

export function useCreateWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateWorkspaceInput) => workspacesApi.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });
}

export function useUpdateMemberRole(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: WorkspaceRole }) =>
      workspacesApi.updateMemberRole(workspaceId, userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'members'] });
    },
  });
}

export function useRemoveMember(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => workspacesApi.removeMember(workspaceId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId, 'members'] });
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
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });
}

export function useUploadWorkspaceIcon(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (file: File) => workspacesApi.uploadIcon(workspaceId, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId] });
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });
}

export function useDeleteWorkspaceIcon(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => workspacesApi.deleteIcon(workspaceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace', workspaceId] });
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });
}

export function useWorkspaceNotifications() {
  const query = useQuery({
    queryKey: ['workspaces', 'notifications'],
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
      await queryClient.cancelQueries({ queryKey: ['auth', 'me'] });

      const previousData = queryClient.getQueryData<{
        user: unknown;
        workspaces: WorkspaceSummary[];
      }>(['auth', 'me']);

      // Optimistically reorder workspaces
      queryClient.setQueryData(
        ['auth', 'me'],
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
        queryClient.setQueryData(['auth', 'me'], context.previousData);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });
}
