import { useState, useMemo, useEffect, useCallback } from 'react';
import { View, Text, FlatList, SectionList, Pressable, TextInput, Modal } from 'react-native';
import { COMMON_EMOJIS, EMOJI_CATEGORIES, searchAllEmojis, useAddReaction } from '@enzyme/shared';

interface ReactionPickerProps {
  visible: boolean;
  messageId: string;
  channelId: string;
  onDismiss: () => void;
}

const SEARCH_CONTENT_STYLE = { padding: 8 };

// Pre-build sections for SectionList (avoids rebuilding on every render)
const EMOJI_SECTIONS = EMOJI_CATEGORIES.map((category) => ({
  title: category.label,
  data: category.emojis,
}));

export function ReactionPicker({ visible, messageId, channelId, onDismiss }: ReactionPickerProps) {
  const [search, setSearch] = useState('');
  const addReaction = useAddReaction(channelId);

  // Reset search when picker closes
  useEffect(() => {
    if (!visible) setSearch('');
  }, [visible]);

  const handleSelect = useCallback(
    (emoji: string) => {
      addReaction.mutate({ messageId, emoji });
      onDismiss();
    },
    [addReaction, messageId, onDismiss],
  );

  const searchResults = useMemo(() => {
    if (!search.trim()) return null;
    return searchAllEmojis(search.trim(), 50, []);
  }, [search]);

  const renderSectionHeader = useCallback(
    ({ section }: { section: { title: string } }) => (
      <Text className="bg-white px-2 pb-1 pt-3 text-xs font-semibold uppercase text-neutral-500 dark:bg-neutral-900 dark:text-neutral-400">
        {section.title}
      </Text>
    ),
    [],
  );

  const renderEmojiItem = useCallback(
    ({ item }: { item: { emoji: string; aliases: string[] } }) => (
      <Pressable className="w-[12.5%] items-center py-2" onPress={() => handleSelect(item.emoji)}>
        <Text className="text-2xl">{item.emoji}</Text>
      </Pressable>
    ),
    [handleSelect],
  );

  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={onDismiss}>
      <Pressable className="flex-1 bg-black/40" onPress={onDismiss} />

      <View className="h-2/3 rounded-t-2xl bg-white dark:bg-neutral-900">
        {/* Header */}
        <View className="items-center py-2">
          <View className="h-1 w-10 rounded-full bg-neutral-300 dark:bg-neutral-600" />
        </View>

        {/* Search */}
        <View className="px-4 pb-2">
          <TextInput
            className="rounded-lg bg-neutral-100 px-3 py-2 text-base text-neutral-900 dark:bg-neutral-800 dark:text-white"
            placeholder="Search emoji..."
            placeholderTextColor="#9ca3af"
            value={search}
            onChangeText={setSearch}
            autoCorrect={false}
          />
        </View>

        {/* Quick reactions */}
        {!search && (
          <View className="flex-row justify-around border-b border-neutral-200 px-4 pb-3 dark:border-neutral-700">
            {COMMON_EMOJIS.slice(0, 6).map((emoji) => (
              <Pressable
                key={emoji}
                className="h-10 w-10 items-center justify-center rounded-lg active:bg-neutral-100 dark:active:bg-neutral-800"
                onPress={() => handleSelect(emoji)}
              >
                <Text className="text-2xl">{emoji}</Text>
              </Pressable>
            ))}
          </View>
        )}

        {/* Emoji grid */}
        {searchResults ? (
          <FlatList
            data={searchResults}
            keyExtractor={(item) => item.shortcode}
            numColumns={8}
            keyboardShouldPersistTaps="handled"
            contentContainerStyle={SEARCH_CONTENT_STYLE}
            renderItem={({ item }) => (
              <Pressable
                className="flex-1 items-center py-2"
                onPress={() => handleSelect(item.emoji ?? `:${item.shortcode}:`)}
              >
                <Text className="text-2xl">{item.emoji ?? `:${item.shortcode}:`}</Text>
              </Pressable>
            )}
          />
        ) : (
          <SectionList
            sections={EMOJI_SECTIONS}
            keyExtractor={(item) => item.aliases[0]}
            keyboardShouldPersistTaps="handled"
            stickySectionHeadersEnabled={false}
            renderSectionHeader={renderSectionHeader}
            renderItem={renderEmojiItem}
            contentContainerStyle={SEARCH_CONTENT_STYLE}
          />
        )}
      </View>
    </Modal>
  );
}
