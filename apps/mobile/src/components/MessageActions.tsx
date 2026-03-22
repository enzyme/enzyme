import { useCallback } from 'react';
import { View, Text, Pressable, Modal, Alert } from 'react-native';
import * as Clipboard from 'expo-clipboard';
import { useDeleteMessage, setEditingMessageId } from '@enzyme/shared';
import type { MessageWithUser } from '@enzyme/api-client';
import { ReactionPicker } from './ReactionPicker';

interface MessageActionsProps {
  message: MessageWithUser | null;
  reactionMessage: MessageWithUser | null;
  onDismiss: () => void;
  onReply?: (messageId: string) => void;
  onShowReactionPicker: (message: MessageWithUser) => void;
  channelId: string;
  currentUserId?: string;
}

export function MessageActions({
  message,
  reactionMessage,
  onDismiss,
  onReply,
  onShowReactionPicker,
  channelId,
  currentUserId,
}: MessageActionsProps) {
  const deleteMessage = useDeleteMessage();
  const activeMessage = message ?? reactionMessage;
  const showActions = !!message;
  const showReactionPicker = !!reactionMessage;

  const isOwnMessage = activeMessage?.user_id === currentUserId;

  const handleReply = useCallback(() => {
    if (!activeMessage || !onReply) return;
    onDismiss();
    onReply(activeMessage.id);
  }, [activeMessage, onDismiss, onReply]);

  const handleAddReaction = useCallback(() => {
    if (!message) return;
    onShowReactionPicker(message);
  }, [message, onShowReactionPicker]);

  const handleCopy = useCallback(async () => {
    if (!activeMessage) return;
    await Clipboard.setStringAsync(activeMessage.content);
    onDismiss();
  }, [activeMessage, onDismiss]);

  const handleEdit = useCallback(() => {
    if (!activeMessage) return;
    setEditingMessageId(activeMessage.id);
    onDismiss();
  }, [activeMessage, onDismiss]);

  const handleDelete = useCallback(() => {
    if (!activeMessage) return;
    Alert.alert('Delete message', 'Are you sure you want to delete this message?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: () => {
          deleteMessage.mutate(activeMessage.id);
          onDismiss();
        },
      },
    ]);
  }, [activeMessage, deleteMessage, onDismiss]);

  return (
    <>
      {/* Action Sheet */}
      <Modal visible={showActions} animationType="fade" transparent onRequestClose={onDismiss}>
        <Pressable className="flex-1 justify-end bg-black/40" onPress={onDismiss}>
          <Pressable className="rounded-t-2xl bg-white pb-8 dark:bg-neutral-800">
            <View className="items-center py-2">
              <View className="h-1 w-10 rounded-full bg-neutral-300 dark:bg-neutral-600" />
            </View>

            {onReply && <ActionButton label="Reply in thread" onPress={handleReply} />}
            <ActionButton label="Add reaction" onPress={handleAddReaction} />
            <ActionButton label="Copy text" onPress={handleCopy} />
            {isOwnMessage && <ActionButton label="Edit" onPress={handleEdit} />}
            {isOwnMessage && <ActionButton label="Delete" onPress={handleDelete} destructive />}
          </Pressable>
        </Pressable>
      </Modal>

      {/* Reaction Picker */}
      {activeMessage && (
        <ReactionPicker
          visible={showReactionPicker}
          messageId={activeMessage.id}
          channelId={channelId}
          onDismiss={onDismiss}
        />
      )}
    </>
  );
}

function ActionButton({
  label,
  onPress,
  destructive = false,
}: {
  label: string;
  onPress: () => void;
  destructive?: boolean;
}) {
  return (
    <Pressable
      className="px-6 py-3.5 active:bg-neutral-100 dark:active:bg-neutral-700"
      onPress={onPress}
    >
      <Text
        className={`text-base ${
          destructive ? 'font-semibold text-red-500' : 'text-neutral-900 dark:text-white'
        }`}
      >
        {label}
      </Text>
    </Pressable>
  );
}
