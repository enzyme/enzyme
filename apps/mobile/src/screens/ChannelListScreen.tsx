import { useCallback, useEffect, useMemo } from 'react';
import { View, Text, SectionList, Pressable, ActivityIndicator } from 'react-native';
import { useChannels, useWorkspace } from '@enzyme/shared';
import type { ChannelWithMembership } from '@enzyme/api-client';
import type { MainScreenProps } from '../navigation/types';
import { UnreadBadge } from '../components/UnreadBadge';
import { Avatar } from '../components/Avatar';
import { useActiveWorkspace } from '../lib/WorkspaceProvider';

type Section = {
  title: string;
  data: ChannelWithMembership[];
};

function channelIcon(channel: ChannelWithMembership): string {
  if (channel.type === 'dm' || channel.type === 'group_dm') return '💬';
  if (channel.type === 'private') return '🔒';
  return '#';
}

export function ChannelListScreen({ route, navigation }: MainScreenProps<'ChannelList'>) {
  const { workspaceId } = route.params;
  const { data: workspaceData } = useWorkspace(workspaceId);
  const { data: channelsData, isLoading, refetch, isRefetching } = useChannels(workspaceId);
  const { setActiveWorkspaceId } = useActiveWorkspace();

  useEffect(() => {
    setActiveWorkspaceId(workspaceId);
    return () => setActiveWorkspaceId(null);
  }, [workspaceId, setActiveWorkspaceId]);

  useEffect(() => {
    if (workspaceData?.workspace?.name) {
      navigation.setOptions({ title: workspaceData.workspace.name });
    }
  }, [workspaceData?.workspace?.name, navigation]);

  const sections = useMemo<Section[]>(() => {
    const channels = channelsData?.channels;
    if (!channels) return [];

    const starred: ChannelWithMembership[] = [];
    const regular: ChannelWithMembership[] = [];
    const dms: ChannelWithMembership[] = [];

    for (const ch of channels) {
      if (ch.archived_at) continue;
      if (ch.is_starred) {
        starred.push(ch);
      } else if (ch.type === 'dm' || ch.type === 'group_dm') {
        dms.push(ch);
      } else {
        regular.push(ch);
      }
    }

    const result: Section[] = [];
    if (starred.length > 0) result.push({ title: 'Starred', data: starred });
    if (regular.length > 0) result.push({ title: 'Channels', data: regular });
    if (dms.length > 0) result.push({ title: 'Direct Messages', data: dms });
    return result;
  }, [channelsData]);

  const renderChannel = useCallback(
    ({ item }: { item: ChannelWithMembership }) => {
      const isDM = item.type === 'dm' || item.type === 'group_dm';
      const dmName =
        isDM && item.dm_participants?.length
          ? item.dm_participants.map((p) => p.display_name).join(', ')
          : null;

      // For 1:1 DMs, show the other participant's avatar with presence
      const dmParticipant =
        item.type === 'dm' && item.dm_participants?.length === 1 ? item.dm_participants[0] : null;

      return (
        <Pressable
          className="flex-row items-center px-4 py-3 active:bg-neutral-100 dark:active:bg-neutral-800"
          onPress={() =>
            navigation.navigate('Channel', {
              workspaceId,
              channelId: item.id,
              channelName: dmName ?? item.name,
            })
          }
        >
          {dmParticipant ? (
            <View className="w-8 items-center">
              <Avatar
                user={{
                  display_name: dmParticipant.display_name,
                  avatar_url: dmParticipant.avatar_url,
                  gravatar_url: dmParticipant.gravatar_url,
                  id: dmParticipant.user_id,
                }}
                size="sm"
                showPresence
              />
            </View>
          ) : (
            <Text className="w-8 text-center text-lg">{channelIcon(item)}</Text>
          )}
          <View className="ml-1 flex-1">
            <Text
              className={`text-base ${item.unread_count > 0 ? 'font-bold' : 'font-normal'} text-neutral-900 dark:text-white`}
              numberOfLines={1}
            >
              {dmName ?? item.name}
            </Text>
          </View>
          {item.unread_count > 0 && <UnreadBadge count={item.unread_count} />}
        </Pressable>
      );
    },
    [navigation, workspaceId],
  );

  const renderSectionHeader = useCallback(
    ({ section }: { section: Section }) => (
      <View className="bg-white px-4 pb-1 pt-4 dark:bg-neutral-900">
        <Text className="text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
          {section.title}
        </Text>
      </View>
    ),
    [],
  );

  if (isLoading) {
    return (
      <View className="flex-1 items-center justify-center bg-white dark:bg-neutral-900">
        <ActivityIndicator size="large" />
      </View>
    );
  }

  return (
    <View className="flex-1 bg-white dark:bg-neutral-900">
      <SectionList
        sections={sections}
        keyExtractor={(item) => item.id}
        renderItem={renderChannel}
        renderSectionHeader={renderSectionHeader}
        onRefresh={refetch}
        refreshing={isRefetching}
        stickySectionHeadersEnabled={false}
        ListEmptyComponent={
          <View className="items-center px-8 pt-20">
            <Text className="text-center text-base text-neutral-500 dark:text-neutral-400">
              No channels yet
            </Text>
          </View>
        }
      />
    </View>
  );
}
