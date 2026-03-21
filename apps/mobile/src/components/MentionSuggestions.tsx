import { useMemo } from 'react';
import { View, Text, FlatList, Pressable } from 'react-native';
import { useWorkspaceMembers, useChannels } from '@enzyme/shared';
import { Avatar } from './Avatar';
import { buildSuggestions, type Suggestion } from '../lib/buildSuggestions';

interface MentionSuggestionsProps {
  text: string;
  cursorPosition: number;
  workspaceId: string;
  onSelect: (token: string) => void;
}

export function MentionSuggestions({
  text,
  cursorPosition,
  workspaceId,
  onSelect,
}: MentionSuggestionsProps) {
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const { data: channelsData } = useChannels(workspaceId);

  const suggestions = useMemo<Suggestion[]>(
    () => buildSuggestions(text, cursorPosition, membersData?.members, channelsData?.channels),
    [text, cursorPosition, membersData, channelsData],
  );

  if (suggestions.length === 0) return null;

  return (
    <View className="max-h-52 border-t border-neutral-200 bg-white dark:border-neutral-700 dark:bg-neutral-900">
      <FlatList
        data={suggestions}
        keyExtractor={(item) => item.id}
        keyboardShouldPersistTaps="handled"
        renderItem={({ item }) => (
          <Pressable
            className="flex-row items-center px-4 py-2.5 active:bg-neutral-100 dark:active:bg-neutral-800"
            onPress={() => onSelect(item.token)}
          >
            {item.avatarUser ? (
              <Avatar user={item.avatarUser} size="sm" />
            ) : (
              <Text className="w-7 text-center text-base">{item.icon}</Text>
            )}
            <Text className="ml-2 text-base text-neutral-900 dark:text-white">
              {item.displayText}
            </Text>
          </Pressable>
        )}
      />
    </View>
  );
}
