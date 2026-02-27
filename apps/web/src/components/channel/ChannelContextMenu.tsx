import {
  StarIcon as StarOutline,
  EnvelopeOpenIcon,
  BellSlashIcon,
  BellIcon,
  LinkIcon,
  ArrowRightStartOnRectangleIcon,
  UserCircleIcon,
} from '@heroicons/react/24/outline';
import { StarIcon as StarSolid } from '@heroicons/react/24/solid';
import { ContextMenu, useContextMenu, MenuItem, MenuSeparator, toast } from '../ui';
import {
  useStarChannel,
  useUnstarChannel,
  useMarkChannelAsRead,
  useLeaveChannel,
} from '../../hooks/useChannels';
import {
  useChannelNotifications,
  useUpdateChannelNotifications,
} from '../../hooks/useChannelNotifications';
import { useProfilePanel } from '../../hooks/usePanel';
import type { ChannelWithMembership } from '@enzyme/api-client';

interface ChannelContextMenuProps {
  channel: ChannelWithMembership;
  workspaceId: string;
  children: (onContextMenu: (e: React.MouseEvent) => void, isMenuOpen: boolean) => React.ReactNode;
}

export function ChannelContextMenu({ channel, workspaceId, children }: ChannelContextMenuProps) {
  const { isOpen, setIsOpen, position, onContextMenu } = useContextMenu();
  const { openProfile } = useProfilePanel();
  const starChannel = useStarChannel(workspaceId);
  const unstarChannel = useUnstarChannel(workspaceId);
  const markAsRead = useMarkChannelAsRead(workspaceId);
  const leaveChannel = useLeaveChannel(workspaceId);
  const { data: notifData } = useChannelNotifications(isOpen ? channel.id : undefined);
  const updateNotifications = useUpdateChannelNotifications(channel.id);

  const isGroupDM = channel.type === 'group_dm';
  const isChannel = channel.type === 'public' || channel.type === 'private';
  const isMuted = notifData?.preferences?.notify_level === 'none';
  const hasUnread = channel.unread_count > 0;
  const canLeave = isChannel ? !channel.is_default : isGroupDM;

  const handleStar = () => {
    if (channel.is_starred) {
      unstarChannel.mutate(channel.id);
    } else {
      starChannel.mutate(channel.id);
    }
  };

  const handleMarkAsRead = () => {
    markAsRead.mutate({ channelId: channel.id });
  };

  const handleToggleMute = () => {
    // Unmute restores to 'all'; a user who previously had 'mentions' can re-set it in settings
    updateNotifications.mutate({
      notify_level: isMuted ? 'all' : 'none',
      email_enabled: notifData?.preferences?.email_enabled ?? true,
    });
  };

  const handleCopyLink = () => {
    const url = `${window.location.origin}/workspaces/${workspaceId}/channels/${channel.id}`;
    navigator.clipboard.writeText(url);
    toast('Link copied to clipboard', 'success');
  };

  const handleLeave = () => {
    leaveChannel.mutate(channel.id);
  };

  return (
    <>
      {children(onContextMenu, isOpen)}
      <ContextMenu isOpen={isOpen} onOpenChange={setIsOpen} position={position}>
        <MenuItem
          onAction={handleStar}
          icon={
            channel.is_starred ? (
              <StarSolid className="h-4 w-4 text-yellow-500" />
            ) : (
              <StarOutline className="h-4 w-4" />
            )
          }
        >
          {channel.is_starred ? 'Unstar' : 'Star'}
        </MenuItem>

        {channel.type === 'dm' && channel.dm_participants?.[0] && (
          <MenuItem
            onAction={() => openProfile(channel.dm_participants![0].user_id)}
            icon={<UserCircleIcon className="h-4 w-4" />}
          >
            View Profile
          </MenuItem>
        )}

        {hasUnread && (
          <MenuItem onAction={handleMarkAsRead} icon={<EnvelopeOpenIcon className="h-4 w-4" />}>
            Mark as Read
          </MenuItem>
        )}

        {isChannel && (
          <MenuItem
            onAction={handleToggleMute}
            icon={
              isMuted ? <BellIcon className="h-4 w-4" /> : <BellSlashIcon className="h-4 w-4" />
            }
          >
            {isMuted ? 'Unmute' : 'Mute'}
          </MenuItem>
        )}

        <MenuItem onAction={handleCopyLink} icon={<LinkIcon className="h-4 w-4" />}>
          Copy Link
        </MenuItem>

        {canLeave && (
          <>
            <MenuSeparator />
            <MenuItem
              onAction={handleLeave}
              variant="danger"
              icon={<ArrowRightStartOnRectangleIcon className="h-4 w-4" />}
            >
              Leave {isChannel ? 'Channel' : 'Conversation'}
            </MenuItem>
          </>
        )}
      </ContextMenu>
    </>
  );
}
