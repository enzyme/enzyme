import { useCallback, useEffect, useRef, useState } from 'react';
import { View, TextInput, Pressable, Text, ScrollView } from 'react-native';
import {
  useSendMessage,
  useSendThreadReply,
  useUpdateMessage,
  useEditingMessageId,
  clearEditingMessageId,
  useMessage,
} from '@enzyme/shared';
import { MentionSuggestions } from './MentionSuggestions';
import {
  FORMAT_ACTIONS,
  applyFormat as applyFormatHelper,
  insertMentionToken,
} from '../lib/applyFormat';

const MAX_LENGTH = 40000;
const WARN_THRESHOLD = 39000;

interface MessageComposerProps {
  channelId: string;
  workspaceId: string;
  threadParentId?: string;
  alsoSendToChannel?: boolean;
  bottomInset?: number;
}

export function MessageComposer({
  channelId,
  workspaceId,
  threadParentId,
  alsoSendToChannel,
  bottomInset = 0,
}: MessageComposerProps) {
  const [text, setText] = useState('');
  const [selection, setSelection] = useState({ start: 0, end: 0 });
  const [showFormatBar, setShowFormatBar] = useState(false);
  const inputRef = useRef<TextInput>(null);

  const editingMessageId = useEditingMessageId();
  const { data: editingMessage } = useMessage(editingMessageId ?? undefined);

  const sendMessage = useSendMessage(channelId);
  const sendThreadReply = useSendThreadReply(threadParentId ?? '', channelId);
  const updateMessage = useUpdateMessage();

  // Load editing message content
  const editLoadedRef = useRef<string | null>(null);
  useEffect(() => {
    if (editingMessageId && editingMessage?.message && editLoadedRef.current !== editingMessageId) {
      setText(editingMessage.message.content);
      editLoadedRef.current = editingMessageId;
    }
  }, [editingMessageId, editingMessage?.message]);

  const handleSend = useCallback(() => {
    const trimmed = text.trim();
    if (!trimmed) return;

    if (editingMessageId) {
      updateMessage.mutate(
        { messageId: editingMessageId, content: trimmed },
        {
          onSuccess: () => {
            setText('');
            editLoadedRef.current = null;
            clearEditingMessageId();
          },
        },
      );
    } else if (threadParentId) {
      sendThreadReply.mutate(
        { content: trimmed, also_send_to_channel: alsoSendToChannel },
        { onSuccess: () => setText('') },
      );
    } else {
      sendMessage.mutate({ content: trimmed }, { onSuccess: () => setText('') });
    }
  }, [
    text,
    editingMessageId,
    threadParentId,
    alsoSendToChannel,
    updateMessage,
    sendThreadReply,
    sendMessage,
  ]);

  const handleCancelEdit = useCallback(() => {
    clearEditingMessageId();
    setText('');
    editLoadedRef.current = null;
  }, []);

  const applyFormat = useCallback(
    (action: (typeof FORMAT_ACTIONS)[number]) => {
      setText(applyFormatHelper(text, selection, action));
    },
    [text, selection],
  );

  const handleMentionSelect = useCallback(
    (token: string) => {
      const result = insertMentionToken(text, selection.start, token);
      if (!result) return;
      setText(result.newText);
      setSelection({ start: result.newPosition, end: result.newPosition });
    },
    [text, selection],
  );

  const isSending = sendMessage.isPending || sendThreadReply.isPending || updateMessage.isPending;
  const canSend = text.trim().length > 0 && !isSending;
  const showCharCount = text.length > WARN_THRESHOLD;

  return (
    <View>
      <MentionSuggestions
        text={text}
        cursorPosition={selection.start}
        workspaceId={workspaceId}
        onSelect={handleMentionSelect}
      />

      {editingMessageId && (
        <View className="flex-row items-center justify-between border-t border-neutral-200 bg-yellow-50 px-4 py-2 dark:border-neutral-700 dark:bg-yellow-900/20">
          <Text className="text-sm font-medium text-yellow-700 dark:text-yellow-400">
            Editing message
          </Text>
          <Pressable onPress={handleCancelEdit}>
            <Text className="text-sm font-medium text-neutral-500 dark:text-neutral-400">
              Cancel
            </Text>
          </Pressable>
        </View>
      )}

      <View className="border-t border-neutral-200 dark:border-neutral-700">
        {/* Formatting toolbar (toggled) */}
        {showFormatBar && (
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={{ paddingHorizontal: 8 }}
            className="border-b border-neutral-100 py-1 dark:border-neutral-800"
          >
            {FORMAT_ACTIONS.map((action) => (
              <Pressable
                key={action.label}
                className="mx-0.5 rounded px-3 py-1 active:bg-neutral-200 dark:active:bg-neutral-700"
                onPress={() => applyFormat(action)}
              >
                <Text className="text-sm font-medium text-neutral-600 dark:text-neutral-400">
                  {action.label}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        )}

        <View className="flex-row items-end px-3 py-2">
          <Pressable
            className={`mr-1.5 h-9 items-center justify-center rounded-lg px-2 ${showFormatBar ? 'bg-neutral-200 dark:bg-neutral-700' : ''}`}
            onPress={() => setShowFormatBar((v) => !v)}
          >
            <Text
              className={`text-sm font-semibold ${showFormatBar ? 'text-blue-500' : 'text-neutral-400 dark:text-neutral-500'}`}
            >
              Aa
            </Text>
          </Pressable>
          <TextInput
            ref={inputRef}
            className="max-h-32 min-h-9 flex-1 rounded-xl bg-neutral-100 px-3 py-2 text-base text-neutral-900 dark:bg-neutral-800 dark:text-white"
            placeholder={editingMessageId ? 'Edit message...' : 'Message...'}
            placeholderTextColor="#9ca3af"
            multiline
            value={text}
            onChangeText={setText}
            onSelectionChange={(e) => setSelection(e.nativeEvent.selection)}
            maxLength={MAX_LENGTH}
          />
          <Pressable
            className={`ml-2 h-9 w-9 items-center justify-center rounded-full ${canSend ? 'bg-blue-500' : 'bg-neutral-300 dark:bg-neutral-700'}`}
            onPress={handleSend}
            disabled={!canSend}
          >
            <Text className="text-base font-bold text-white">↑</Text>
          </Pressable>
        </View>

        {showCharCount && (
          <Text className="px-4 pb-1 text-xs text-neutral-400">
            {text.length.toLocaleString()}/{MAX_LENGTH.toLocaleString()}
          </Text>
        )}
      </View>
      {bottomInset > 0 && <View style={{ height: bottomInset }} />}
    </View>
  );
}
