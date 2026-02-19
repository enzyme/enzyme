import { Cog6ToothIcon, UserPlusIcon, EnvelopeOpenIcon } from '@heroicons/react/24/outline';
import { ContextMenu, useContextMenu, MenuItem, MenuSeparator } from '../ui';
import { useMarkAllChannelsAsRead } from '../../hooks/useChannels';

interface WorkspaceContextMenuProps {
  workspaceId: string;
  onOpenWorkspaceSettings: () => void;
  onOpenInvite: () => void;
  canInvite?: boolean;
  children: (onContextMenu: (e: React.MouseEvent) => void, isMenuOpen: boolean) => React.ReactNode;
}

export function WorkspaceContextMenu({
  workspaceId,
  onOpenWorkspaceSettings,
  onOpenInvite,
  canInvite,
  children,
}: WorkspaceContextMenuProps) {
  const { isOpen, setIsOpen, position, onContextMenu } = useContextMenu();
  const markAllAsRead = useMarkAllChannelsAsRead(workspaceId);

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
      </ContextMenu>
    </>
  );
}
