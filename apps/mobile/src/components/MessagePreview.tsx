import { View, Text, Pressable } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { formatRelativeTime } from '@enzyme/shared';
import type {
  LinkPreview,
  WorkspaceMemberWithUser,
  ChannelWithMembership,
} from '@enzyme/api-client';
import type { MainStackParamList } from '../navigation/types';
import { Avatar } from './Avatar';
import { MrkdwnRenderer } from './MrkdwnRenderer';

interface MessagePreviewProps {
  preview: LinkPreview;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  workspaceId: string;
}

export function MessagePreview({ preview, members, channels, workspaceId }: MessagePreviewProps) {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();

  const isInaccessible = preview.linked_channel_type === 'inaccessible';
  const isDeleted = preview.linked_channel_type === 'deleted';

  function handlePress() {
    if (isInaccessible || isDeleted || !preview.linked_channel_id) return;
    navigation.navigate('Channel', {
      workspaceId,
      channelId: preview.linked_channel_id,
      channelName: preview.linked_channel_name ?? '',
      scrollToMessageId: preview.linked_message_id,
    });
  }

  if (isInaccessible || isDeleted) {
    return (
      <View className="mt-2 flex-row items-center rounded-lg border border-neutral-200 bg-neutral-50 px-3 py-2.5 dark:border-neutral-700 dark:bg-neutral-800/50">
        <Text className="text-sm text-neutral-400 dark:text-neutral-500">
          {isInaccessible ? '🔒' : '🗑'}
        </Text>
        <Text className="ml-2 text-sm italic text-neutral-500 dark:text-neutral-400">
          {isInaccessible ? 'Message from a private channel' : 'This message was deleted'}
        </Text>
      </View>
    );
  }

  return (
    <Pressable
      className="mt-2 overflow-hidden rounded-lg border border-neutral-200 bg-white active:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:active:bg-neutral-700"
      onPress={handlePress}
    >
      <View className="flex-row">
        {/* Blue left border accent */}
        <View className="w-1 bg-blue-500" />
        <View className="min-w-0 flex-1 px-3 py-2">
          {/* Header: avatar + author + channel + time */}
          <View className="flex-row items-center" style={{ gap: 6 }}>
            <Avatar
              user={{
                display_name: preview.message_author_name ?? 'Unknown',
                avatar_url: preview.message_author_avatar_url,
                id: preview.message_author_id,
              }}
              size="sm"
            />
            <Text
              className="text-sm font-medium text-neutral-900 dark:text-white"
              numberOfLines={1}
            >
              {preview.message_author_name ?? 'Unknown'}
            </Text>
            {preview.linked_channel_name && (
              <>
                <Text className="text-xs text-neutral-400 dark:text-neutral-500">in</Text>
                <Text
                  className="text-xs font-medium text-neutral-500 dark:text-neutral-400"
                  numberOfLines={1}
                >
                  #{preview.linked_channel_name}
                </Text>
              </>
            )}
            {preview.message_created_at && (
              <Text className="text-xs text-neutral-400 dark:text-neutral-500">
                {formatRelativeTime(preview.message_created_at)}
              </Text>
            )}
          </View>
          {/* Message content */}
          {preview.message_content && (
            <View className="mt-1">
              <MrkdwnRenderer
                content={preview.message_content}
                members={members}
                channels={channels}
              />
            </View>
          )}
        </View>
      </View>
    </Pressable>
  );
}
