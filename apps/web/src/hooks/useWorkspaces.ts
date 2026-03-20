import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { toast } from '../components/ui';
import type { WorkspaceSummary } from '@enzyme/api-client';
import { useLeaveWorkspace } from '@enzyme/shared';

// Re-export all pure hooks from shared
export {
  useWorkspace,
  useWorkspaceMembers,
  useCreateWorkspace,
  useUpdateWorkspace,
  useUpdateMemberRole,
  useRemoveMember,
  useLeaveWorkspace,
  useCreateInvite,
  useAcceptInvite,
  useUploadWorkspaceIcon,
  useDeleteWorkspaceIcon,
  useWorkspaceNotifications,
  useReorderWorkspaces,
} from '@enzyme/shared';

// Web-specific: combines leave with navigation and toast
export function useLeaveAndNavigate(workspaces: WorkspaceSummary[] | undefined) {
  const leaveWorkspace = useLeaveWorkspace();
  const navigate = useNavigate();

  const leave = useCallback(
    async (workspaceId: string) => {
      const otherWorkspaces = workspaces?.filter((ws) => ws.id !== workspaceId && !ws.ban) ?? [];
      const redirectTo = otherWorkspaces.length > 0 ? `/workspaces/${otherWorkspaces[0].id}` : '/';

      try {
        await leaveWorkspace.mutateAsync(workspaceId);
        navigate(redirectTo);
      } catch {
        toast('Failed to leave workspace', 'error');
      }
    },
    [workspaces, leaveWorkspace, navigate],
  );

  return { leave, isPending: leaveWorkspace.isPending };
}
