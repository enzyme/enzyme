import { useCallback, useMemo } from 'react';
import { View, Text, FlatList, Pressable, ActivityIndicator, Alert } from 'react-native';
import {
  useChannels,
  useChannelMembers,
  useWorkspaceMembers,
  useAddChannelMember,
  useAuth,
} from '@enzyme/shared';
import type { ChannelMember, WorkspaceMemberWithUser } from '@enzyme/api-client';
import type { MainScreenProps } from '../navigation/types';
import { Avatar } from '../components/ui/Avatar';
import { formatDate } from '@enzyme/shared';

type ListItem =
  | { type: 'member'; data: ChannelMember }
  | { type: 'addHeader' }
  | { type: 'nonMember'; data: WorkspaceMemberWithUser };

export function ChannelDetailsScreen({ route, navigation }: MainScreenProps<'ChannelDetails'>) {
  const { workspaceId, channelId } = route.params;
  const { workspaces } = useAuth();
  const { data: channelsData } = useChannels(workspaceId);
  const { data: membersData, isLoading: membersLoading } = useChannelMembers(channelId);
  const { data: workspaceMembersData } = useWorkspaceMembers(workspaceId);
  const addMember = useAddChannelMember(channelId);

  const channel = channelsData?.channels?.find((c) => c.id === channelId);
  const isDM = channel?.type === 'dm' || channel?.type === 'group_dm';
  const channelMembers: ChannelMember[] = membersData?.members ?? [];

  const workspaceRole = workspaces?.find((w) => w.id === workspaceId)?.role;
  const canAddMembers =
    workspaceRole === 'admin' || workspaceRole === 'owner' || channel?.channel_role !== undefined;

  const nonMembers = useMemo(() => {
    if (!canAddMembers || !workspaceMembersData?.members || !membersData?.members) return [];
    const memberIds = new Set(membersData.members.map((m: ChannelMember) => m.user_id));
    return workspaceMembersData.members.filter((m) => !memberIds.has(m.user_id));
  }, [canAddMembers, workspaceMembersData, membersData]);

  const handleAddMember = useCallback(
    (userId: string, displayName: string) => {
      Alert.alert('Add member', `Add ${displayName} to #${channel?.name}?`, [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Add',
          onPress: () => addMember.mutate({ userId }),
        },
      ]);
    },
    [channel?.name, addMember],
  );

  const handleMemberPress = useCallback(
    (userId: string) => navigation.navigate('Profile', { workspaceId, userId }),
    [navigation, workspaceId],
  );

  if (!channel) {
    return (
      <View className="flex-1 items-center justify-center bg-white dark:bg-neutral-900">
        <ActivityIndicator size="large" />
      </View>
    );
  }

  const typeLabel = channel.type === 'private' ? 'Private channel' : 'Public channel';
  const typeIcon = channel.type === 'private' ? '🔒' : '#';

  const listData = useMemo<ListItem[]>(() => {
    const items: ListItem[] = channelMembers.map((m) => ({ type: 'member' as const, data: m }));
    if (!isDM && nonMembers.length > 0) {
      items.push({ type: 'addHeader' as const });
      items.push(...nonMembers.map((m) => ({ type: 'nonMember' as const, data: m })));
    }
    return items;
  }, [channelMembers, nonMembers, isDM]);

  const renderItem = useCallback(
    ({ item }: { item: ListItem }) => {
      if (item.type === 'addHeader') {
        return (
          <View className="border-t border-neutral-200 px-4 pb-1 pt-4 dark:border-neutral-700">
            <Text className="text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
              Add Members
            </Text>
          </View>
        );
      }

      if (item.type === 'nonMember') {
        const m = item.data;
        return (
          <Pressable
            className="flex-row items-center px-4 py-2.5 active:bg-neutral-100 dark:active:bg-neutral-800"
            onPress={() => handleAddMember(m.user_id, m.display_name)}
          >
            <Avatar
              user={{
                display_name: m.display_name,
                avatar_url: m.avatar_url,
                gravatar_url: m.gravatar_url,
                id: m.user_id,
              }}
              size="md"
            />
            <Text className="ml-3 flex-1 text-base text-neutral-900 dark:text-white">
              {m.display_name}
            </Text>
            <View className="rounded bg-blue-500 px-3 py-1">
              <Text className="text-sm font-medium text-white">Add</Text>
            </View>
          </Pressable>
        );
      }

      const member = item.data;
      return (
        <Pressable
          className="flex-row items-center px-4 py-2.5 active:bg-neutral-100 dark:active:bg-neutral-800"
          onPress={() => handleMemberPress(member.user_id)}
        >
          <Avatar
            user={{
              display_name: member.display_name,
              avatar_url: member.avatar_url,
              gravatar_url: member.gravatar_url,
              id: member.user_id,
            }}
            size="md"
            showPresence
          />
          <View className="ml-3 flex-1">
            <Text className="text-base text-neutral-900 dark:text-white">
              {member.display_name}
            </Text>
          </View>
          {member.channel_role === 'admin' && (
            <View className="rounded bg-blue-100 px-2 py-0.5 dark:bg-blue-900">
              <Text className="text-xs font-medium text-blue-700 dark:text-blue-300">Admin</Text>
            </View>
          )}
          {member.channel_role === 'poster' && (
            <View className="rounded bg-neutral-100 px-2 py-0.5 dark:bg-neutral-700">
              <Text className="text-xs font-medium text-neutral-600 dark:text-neutral-300">
                Poster
              </Text>
            </View>
          )}
        </Pressable>
      );
    },
    [handleMemberPress, handleAddMember],
  );

  return (
    <FlatList
      className="flex-1 bg-white dark:bg-neutral-900"
      data={listData}
      keyExtractor={(item) => (item.type === 'addHeader' ? 'add-header' : item.data.user_id)}
      ListHeaderComponent={
        <>
          {/* About section (not for DMs) */}
          {!isDM && (
            <View className="border-b border-neutral-200 px-4 py-4 dark:border-neutral-700">
              <Text className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
                About
              </Text>
              <Text className="text-lg font-bold text-neutral-900 dark:text-white">
                {typeIcon} {channel.name}
              </Text>
              {channel.description ? (
                <Text className="mt-1 text-sm text-neutral-600 dark:text-neutral-300">
                  {channel.description}
                </Text>
              ) : null}
              <View className="mt-2 flex-row items-center">
                <Text className="text-sm text-neutral-500 dark:text-neutral-400">{typeLabel}</Text>
                <Text className="mx-2 text-neutral-300 dark:text-neutral-600">·</Text>
                <Text className="text-sm text-neutral-500 dark:text-neutral-400">
                  Created {formatDate(channel.created_at)}
                </Text>
              </View>
            </View>
          )}

          {/* Members header */}
          <View className="px-4 pb-1 pt-4">
            <Text className="text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
              Members ({channelMembers.length})
            </Text>
          </View>

          {membersLoading && (
            <View className="items-center py-4">
              <ActivityIndicator />
            </View>
          )}
        </>
      }
      renderItem={renderItem}
    />
  );
}
