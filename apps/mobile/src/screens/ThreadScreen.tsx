import { useCallback, useMemo, useState } from 'react';
import {
  View,
  FlatList,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Switch,
  Text,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import {
  useMessage,
  useThreadMessages,
  useWorkspaceMembers,
  useChannels,
  useAuth,
} from '@enzyme/shared';
import type { MessageWithUser } from '@enzyme/api-client';
import type { MainScreenProps } from '../navigation/types';
import { MessageBubble } from '../components/MessageBubble';
import { MessageComposer } from '../components/MessageComposer';
import { MessageActions } from '../components/MessageActions';
import { FullScreenLoader } from '../components/FullScreenLoader';
import { ImageViewer } from '../components/ImageViewer';
import { useImageViewer } from '../hooks/useImageViewer';
import { shouldGroupMessages } from '../lib/buildListItems';

const threadKeyExtractor = (item: MessageWithUser) => item.id;
const CONTENT_STYLE = { paddingVertical: 8 };

export function ThreadScreen({ route, navigation }: MainScreenProps<'Thread'>) {
  const { workspaceId, channelId, parentMessageId } = route.params;
  const { user } = useAuth();
  const insets = useSafeAreaInsets();
  const { data: parentData } = useMessage(parentMessageId);
  const parentMessage = parentData?.message;
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useThreadMessages(parentMessageId);
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const { data: channelsData } = useChannels(workspaceId);
  const members = membersData?.members;
  const channels = channelsData?.channels;

  const [actionMessage, setActionMessage] = useState<MessageWithUser | null>(null);
  const [reactionMessage, setReactionMessage] = useState<MessageWithUser | null>(null);
  const [alsoSendToChannel, setAlsoSendToChannel] = useState(false);
  const { viewer, openViewer, closeViewer } = useImageViewer();

  const handleDismissActions = useCallback(() => {
    setActionMessage(null);
    setReactionMessage(null);
  }, []);

  const handleShowReactionPicker = useCallback((msg: MessageWithUser) => {
    setActionMessage(null);
    setReactionMessage(msg);
  }, []);

  const replies = useMemo(
    () => (data?.pages.flatMap((p) => p.messages) ?? []).slice().reverse(),
    [data?.pages],
  );

  const handleEndReached = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const handleAvatarPress = useCallback(
    (userId: string) => navigation.navigate('Profile', { workspaceId, userId }),
    [navigation, workspaceId],
  );

  const renderItem = useCallback(
    ({ item, index }: { item: MessageWithUser; index: number }) => {
      const isGrouped = shouldGroupMessages(item, replies[index + 1]);

      return (
        <MessageBubble
          message={item}
          channelId={channelId}
          workspaceId={workspaceId}
          members={members}
          channels={channels}
          currentUserId={user?.id}
          isGrouped={isGrouped}
          onAvatarPress={handleAvatarPress}
          onLongPress={setActionMessage}
          onReactionPress={setReactionMessage}
          onImagePress={openViewer}
        />
      );
    },
    [channelId, workspaceId, members, channels, user?.id, handleAvatarPress, openViewer, replies],
  );

  if (isLoading) {
    return <FullScreenLoader />;
  }

  return (
    <KeyboardAvoidingView
      className="flex-1 bg-white dark:bg-neutral-900"
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={Platform.OS === 'ios' ? insets.top + 44 : 0}
    >
      {/* Parent message */}
      {parentMessage && (
        <View className="border-b border-neutral-200 bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800/50">
          <MessageBubble
            message={parentMessage}
            channelId={channelId}
            workspaceId={workspaceId}
            members={members}
            channels={channels}
            currentUserId={user?.id}
            onAvatarPress={handleAvatarPress}
            onImagePress={openViewer}
          />
        </View>
      )}

      {/* Replies */}
      <FlatList
        data={replies}
        renderItem={renderItem}
        keyExtractor={threadKeyExtractor}
        inverted
        onEndReached={handleEndReached}
        onEndReachedThreshold={0.5}
        keyboardDismissMode="interactive"
        keyboardShouldPersistTaps="handled"
        windowSize={7}
        maxToRenderPerBatch={10}
        removeClippedSubviews={Platform.OS !== 'ios'}
        ListFooterComponent={
          isFetchingNextPage ? <ActivityIndicator style={{ padding: 16 }} /> : null
        }
        contentContainerStyle={CONTENT_STYLE}
      />

      {/* Also send to channel toggle */}
      <View className="flex-row items-center justify-between border-t border-neutral-200 px-4 py-2 dark:border-neutral-700">
        <Text className="text-sm text-neutral-600 dark:text-neutral-400">Also send to channel</Text>
        <Switch
          value={alsoSendToChannel}
          onValueChange={setAlsoSendToChannel}
          trackColor={{ true: '#3b82f6' }}
        />
      </View>

      <MessageComposer
        channelId={channelId}
        workspaceId={workspaceId}
        threadParentId={parentMessageId}
        alsoSendToChannel={alsoSendToChannel}
        bottomInset={insets.bottom}
      />

      <MessageActions
        message={actionMessage}
        reactionMessage={reactionMessage}
        onDismiss={handleDismissActions}
        onShowReactionPicker={handleShowReactionPicker}
        channelId={channelId}
        currentUserId={user?.id}
      />

      {viewer && (
        <ImageViewer images={viewer.images} initialIndex={viewer.index} onClose={closeViewer} />
      )}
    </KeyboardAvoidingView>
  );
}
