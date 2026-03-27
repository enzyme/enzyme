import { useCallback } from 'react';
import { Alert } from 'react-native';
import {
  useStarChannel,
  useUnstarChannel,
  useMarkChannelAsRead,
  useLeaveChannel,
  useChannelNotifications,
  useUpdateChannelNotifications,
} from '@enzyme/shared';
import type { ChannelWithMembership } from '@enzyme/api-client';
import { BottomSheet } from './ui/BottomSheet';
import { ActionButton } from './ui/ActionButton';

interface ChannelActionsProps {
  channel: ChannelWithMembership | null;
  workspaceId: string;
  onDismiss: () => void;
  onLeave?: () => void;
}

export function ChannelActions({ channel, workspaceId, onDismiss, onLeave }: ChannelActionsProps) {
  const starChannel = useStarChannel(workspaceId);
  const unstarChannel = useUnstarChannel(workspaceId);
  const markAsRead = useMarkChannelAsRead(workspaceId);
  const leaveChannel = useLeaveChannel(workspaceId);
  const { data: notifData } = useChannelNotifications(channel?.id);
  const updateNotifications = useUpdateChannelNotifications(channel?.id ?? '');

  const isDM = channel?.type === 'dm' || channel?.type === 'group_dm';
  const isMuted = notifData?.preferences?.notify_level === 'none';

  const handleStar = useCallback(() => {
    if (!channel) return;
    if (channel.is_starred) {
      unstarChannel.mutate(channel.id);
    } else {
      starChannel.mutate(channel.id);
    }
    onDismiss();
  }, [channel, starChannel, unstarChannel, onDismiss]);

  const handleMute = useCallback(() => {
    if (!channel) return;
    const currentEmailEnabled = notifData?.preferences?.email_enabled ?? true;
    updateNotifications.mutate({
      notify_level: isMuted ? 'all' : 'none',
      email_enabled: currentEmailEnabled,
    });
    onDismiss();
  }, [channel, isMuted, notifData, updateNotifications, onDismiss]);

  const handleMarkAsRead = useCallback(() => {
    if (!channel) return;
    markAsRead.mutate({ channelId: channel.id });
    onDismiss();
  }, [channel, markAsRead, onDismiss]);

  const handleLeave = useCallback(() => {
    if (!channel) return;
    Alert.alert('Leave channel', `Are you sure you want to leave #${channel.name}?`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Leave',
        style: 'destructive',
        onPress: () => {
          leaveChannel.mutate(channel.id);
          onDismiss();
          onLeave?.();
        },
      },
    ]);
  }, [channel, leaveChannel, onDismiss, onLeave]);

  return (
    <BottomSheet visible={!!channel} onDismiss={onDismiss}>
      <ActionButton label={channel?.is_starred ? 'Unstar' : 'Star'} onPress={handleStar} />
      <ActionButton label={isMuted ? 'Unmute' : 'Mute'} onPress={handleMute} />
      {channel && channel.unread_count > 0 && (
        <ActionButton label="Mark as read" onPress={handleMarkAsRead} />
      )}
      {channel && !channel.is_default && !isDM && (
        <ActionButton label="Leave channel" onPress={handleLeave} destructive />
      )}
    </BottomSheet>
  );
}
