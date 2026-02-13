import { Button as AriaButton } from 'react-aria-components';
import { AvatarStack, Spinner } from '../ui';
import { useChannelMembers } from '../../hooks/useChannels';
import type { ChannelType } from '@feather/api-client';

interface ChannelMembersButtonProps {
  channelId: string;
  channelType: ChannelType;
  onPress: () => void;
}

export function ChannelMembersButton({
  channelId,
  channelType,
  onPress,
}: ChannelMembersButtonProps) {
  const { data, isLoading } = useChannelMembers(channelId);

  // Hide for 1:1 DM channels only
  if (channelType === 'dm') {
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
    <AriaButton
      onPress={onPress}
      className="flex cursor-pointer items-center gap-1.5 rounded px-2 py-1 outline-none hover:bg-gray-100 focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-gray-800"
      aria-label={`View ${members.length} channel members`}
    >
      <AvatarStack users={avatarUsers} max={3} size="xs" showCount={false} />
      <span className="text-sm text-gray-600 dark:text-gray-400">{members.length}</span>
    </AriaButton>
  );
}
