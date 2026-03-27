import { useCallback } from 'react';
import { Alert } from 'react-native';
import * as Clipboard from 'expo-clipboard';
import { useDeleteMessage, setEditingMessageId } from '@enzyme/shared';
import type { MessageWithUser } from '@enzyme/api-client';
import { ReactionPicker } from './ReactionPicker';
import { BottomSheet } from './ui/BottomSheet';
import { ActionButton } from './ui/ActionButton';

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
      <BottomSheet visible={showActions} onDismiss={onDismiss}>
        {onReply && <ActionButton label="Reply in thread" onPress={handleReply} />}
        <ActionButton label="Add reaction" onPress={handleAddReaction} />
        <ActionButton label="Copy text" onPress={handleCopy} />
        {isOwnMessage && <ActionButton label="Edit" onPress={handleEdit} />}
        {isOwnMessage && <ActionButton label="Delete" onPress={handleDelete} destructive />}
      </BottomSheet>

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
