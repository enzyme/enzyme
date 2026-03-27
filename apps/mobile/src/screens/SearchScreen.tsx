import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  Pressable,
  ActivityIndicator,
  useColorScheme,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useSearch, formatRelativeTime } from '@enzyme/shared';
import type { SearchMessage } from '@enzyme/api-client';
import type { MainScreenProps } from '../navigation/types';
import { Avatar } from '../components/ui/Avatar';
import { SearchFilters } from '../components/SearchFilters';

export function SearchScreen({ route, navigation }: MainScreenProps<'Search'>) {
  const { workspaceId } = route.params;
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';

  const [queryText, setQueryText] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [offset, setOffset] = useState(0);
  const [allMessages, setAllMessages] = useState<SearchMessage[]>([]);
  const [channelId, setChannelId] = useState<string | undefined>();
  const [userId, setUserId] = useState<string | undefined>();
  const [showFilters, setShowFilters] = useState(false);

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const { data, isLoading, isFetching } = useSearch({
    workspaceId,
    query: debouncedQuery,
    channelId,
    userId,
    offset,
    limit: 20,
  });

  useEffect(() => {
    if (data?.messages) {
      if (offset === 0) {
        setAllMessages(data.messages);
      } else {
        setAllMessages((prev) => [...prev, ...data.messages]);
      }
    }
  }, [data?.messages, offset]);

  const handleTextChange = useCallback((text: string) => {
    setQueryText(text);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setDebouncedQuery(text.trim());
      setOffset(0);
    }, 300);
  }, []);

  const handleLoadMore = useCallback(() => {
    if (data?.has_more && !isFetching) {
      setOffset((prev) => prev + 20);
    }
  }, [data?.has_more, isFetching]);

  const filterCount = (channelId ? 1 : 0) + (userId ? 1 : 0);

  const renderResult = useCallback(
    ({ item }: { item: SearchMessage }) => {
      const channelIcon = item.channel_type === 'private' ? '🔒' : '#';

      return (
        <Pressable
          className="border-b border-neutral-100 px-4 py-3 active:bg-neutral-50 dark:border-neutral-800 dark:active:bg-neutral-800/50"
          onPress={() =>
            navigation.navigate('Channel', {
              workspaceId,
              channelId: item.channel_id,
              channelName: item.channel_name,
              scrollToMessageId: item.id,
            })
          }
        >
          <View className="flex-row items-center">
            <Text className="text-xs text-neutral-500 dark:text-neutral-400">
              {channelIcon} {item.channel_name}
            </Text>
            <Text className="mx-1.5 text-neutral-300 dark:text-neutral-600">·</Text>
            <Text className="text-xs text-neutral-400 dark:text-neutral-500">
              {formatRelativeTime(item.created_at)}
            </Text>
          </View>
          <View className="mt-1.5 flex-row items-start">
            <Avatar
              user={{
                display_name: item.user_display_name ?? '',
                avatar_url: item.user_avatar_url,
                gravatar_url: item.user_gravatar_url,
                id: item.user_id,
              }}
              size="sm"
            />
            <View className="ml-2 flex-1">
              <Text className="text-sm font-semibold text-neutral-900 dark:text-white">
                {item.user_display_name}
              </Text>
              <Text
                className="mt-0.5 text-sm text-neutral-600 dark:text-neutral-300"
                numberOfLines={2}
              >
                {item.content}
              </Text>
            </View>
          </View>
        </Pressable>
      );
    },
    [navigation, workspaceId],
  );

  return (
    <View className="flex-1 bg-white dark:bg-neutral-900">
      {/* Search bar */}
      <View className="flex-row items-center border-b border-neutral-200 px-4 py-2 dark:border-neutral-700">
        <Ionicons name="search-outline" size={20} color={isDark ? '#a3a3a3' : '#737373'} />
        <TextInput
          className="ml-2 flex-1 text-base text-neutral-900 dark:text-white"
          placeholder="Search messages..."
          placeholderTextColor={isDark ? '#737373' : '#a3a3a3'}
          value={queryText}
          onChangeText={handleTextChange}
          autoFocus
          returnKeyType="search"
        />
        <Pressable onPress={() => setShowFilters(true)} className="relative ml-2">
          <Ionicons
            name="options-outline"
            size={22}
            color={filterCount > 0 ? '#3b82f6' : isDark ? '#a3a3a3' : '#737373'}
          />
          {filterCount > 0 && (
            <View className="absolute -right-1 -top-1 h-4 w-4 items-center justify-center rounded-full bg-blue-500">
              <Text className="text-[10px] font-bold text-white">{filterCount}</Text>
            </View>
          )}
        </Pressable>
      </View>

      {/* Results */}
      {isLoading && debouncedQuery ? (
        <View className="items-center pt-20">
          <ActivityIndicator size="large" />
        </View>
      ) : debouncedQuery && allMessages.length === 0 && !isFetching ? (
        <View className="items-center px-8 pt-20">
          <Text className="text-center text-base text-neutral-500 dark:text-neutral-400">
            No results found
          </Text>
        </View>
      ) : !debouncedQuery ? (
        <View className="items-center px-8 pt-20">
          <Ionicons name="search" size={48} color={isDark ? '#525252' : '#d4d4d4'} />
          <Text className="mt-4 text-center text-base text-neutral-500 dark:text-neutral-400">
            Search for messages across channels
          </Text>
        </View>
      ) : (
        <FlatList
          data={allMessages}
          keyExtractor={(item) => item.id}
          renderItem={renderResult}
          onEndReached={handleLoadMore}
          onEndReachedThreshold={0.5}
          ListFooterComponent={
            isFetching ? (
              <View className="items-center py-4">
                <ActivityIndicator />
              </View>
            ) : null
          }
        />
      )}

      <SearchFilters
        visible={showFilters}
        workspaceId={workspaceId}
        selectedChannelId={channelId}
        selectedUserId={userId}
        onSelectChannel={setChannelId}
        onSelectUser={setUserId}
        onDismiss={() => setShowFilters(false)}
      />
    </View>
  );
}
