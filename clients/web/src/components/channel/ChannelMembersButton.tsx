import { useState } from 'react';
import { Button as AriaButton } from 'react-aria-components';
import { AvatarStack, Spinner } from '../ui';
import { ChannelMembersModal } from './ChannelMembersModal';
import { useChannelMembers } from '../../hooks/useChannels';
import type { ChannelType } from '@feather/api-client';

interface ChannelMembersButtonProps {
  channelId: string;
  workspaceId: string;
  channelType: ChannelType;
  canAddMembers: boolean;
}

export function ChannelMembersButton({
  channelId,
  workspaceId,
  channelType,
  canAddMembers,
}: ChannelMembersButtonProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const { data, isLoading } = useChannelMembers(channelId);

  // Hide for DM channels
  if (channelType === 'dm' || channelType === 'group_dm') {
    return null;
  }

  const members = data?.members || [];

  if (isLoading) {
    return (
      <div className="flex items-center px-2">
        <Spinner size="sm" />
      </div>
    );
  }

  if (members.length === 0) {
    return null;
  }

  // Transform members to the format AvatarStack expects
  const avatarUsers = members.map((m) => ({
    user_id: m.user_id,
    display_name: m.display_name,
    avatar_url: m.avatar_url,
  }));

  return (
    <>
      <AriaButton
        onPress={() => setIsModalOpen(true)}
        className="flex items-center gap-1.5 px-2 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
        aria-label={`View ${members.length} channel members`}
      >
        <AvatarStack users={avatarUsers} max={3} size="xs" showCount={false} />
        <span className="text-sm text-gray-600 dark:text-gray-400">{members.length}</span>
      </AriaButton>

      <ChannelMembersModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        channelId={channelId}
        workspaceId={workspaceId}
        canAddMembers={canAddMembers}
      />
    </>
  );
}
