import { useState } from 'react';
import {
  Cog6ToothIcon,
  UserPlusIcon,
  EnvelopeOpenIcon,
  ArrowRightStartOnRectangleIcon,
} from '@heroicons/react/24/outline';
import { ContextMenu, useContextMenu, MenuItem, MenuSeparator, Modal, Button } from '../ui';
import { useMarkAllChannelsAsRead } from '../../hooks/useChannels';
import { useLeaveAndNavigate } from '../../hooks/useWorkspaces';
import { useAuth } from '../../hooks';
import type { WorkspaceRole } from '@enzyme/api-client';

interface WorkspaceContextMenuProps {
  workspaceId: string;
  workspaceName: string;
  workspaceRole: WorkspaceRole;
  onOpenWorkspaceSettings: () => void;
  onOpenInvite: () => void;
  canInvite?: boolean;
  children: (onContextMenu: (e: React.MouseEvent) => void, isMenuOpen: boolean) => React.ReactNode;
}

export function WorkspaceContextMenu({
  workspaceId,
  workspaceName,
  workspaceRole,
  onOpenWorkspaceSettings,
  onOpenInvite,
  canInvite,
  children,
}: WorkspaceContextMenuProps) {
  const { isOpen, setIsOpen, position, onContextMenu } = useContextMenu();
  const markAllAsRead = useMarkAllChannelsAsRead(workspaceId);
  const { workspaces } = useAuth();
  const { leave, isPending } = useLeaveAndNavigate(workspaces);
  const [showLeaveConfirm, setShowLeaveConfirm] = useState(false);

  const isOwner = workspaceRole === 'owner';

  const handleLeave = async () => {
    await leave(workspaceId);
    setShowLeaveConfirm(false);
  };

  return (
    <>
      {children(onContextMenu, isOpen)}
      <ContextMenu isOpen={isOpen} onOpenChange={setIsOpen} position={position}>
        <MenuItem onAction={onOpenWorkspaceSettings} icon={<Cog6ToothIcon className="h-4 w-4" />}>
          Workspace Settings
        </MenuItem>
        {canInvite && (
          <MenuItem onAction={onOpenInvite} icon={<UserPlusIcon className="h-4 w-4" />}>
            Invite People
          </MenuItem>
        )}
        <MenuSeparator />
        <MenuItem
          onAction={() => markAllAsRead.mutate()}
          icon={<EnvelopeOpenIcon className="h-4 w-4" />}
        >
          Mark All as Read
        </MenuItem>
        {!isOwner && (
          <>
            <MenuSeparator />
            <MenuItem
              onAction={() => setShowLeaveConfirm(true)}
              icon={<ArrowRightStartOnRectangleIcon className="h-4 w-4" />}
              variant="danger"
            >
              Leave Workspace
            </MenuItem>
          </>
        )}
      </ContextMenu>

      <Modal
        isOpen={showLeaveConfirm}
        onClose={() => setShowLeaveConfirm(false)}
        title={`Leave ${workspaceName}?`}
        size="sm"
      >
        <p className="mb-5 text-sm text-gray-600 dark:text-gray-300">
          You will lose access to all channels in this workspace. You can rejoin later with an
          invite.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onPress={() => setShowLeaveConfirm(false)}>
            Cancel
          </Button>
          <Button variant="danger" onPress={handleLeave} isLoading={isPending}>
            Leave Workspace
          </Button>
        </div>
      </Modal>
    </>
  );
}
