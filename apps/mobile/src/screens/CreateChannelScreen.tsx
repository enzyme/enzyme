import { useState } from 'react';
import { View, Text, TextInput, Pressable, useColorScheme, ActivityIndicator } from 'react-native';
import { useCreateChannel, CHANNEL_NAME_REGEX } from '@enzyme/shared';
import type { MainScreenProps } from '../navigation/types';

export function CreateChannelScreen({ route, navigation }: MainScreenProps<'CreateChannel'>) {
  const { workspaceId } = route.params;
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';
  const createChannel = useCreateChannel(workspaceId);

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [type, setType] = useState<'public' | 'private'>('public');
  const [error, setError] = useState('');

  const handleNameChange = (text: string) => {
    setName(text.toLowerCase().replace(/\s/g, '-'));
    setError('');
  };

  const isValid = name.length > 0 && CHANNEL_NAME_REGEX.test(name);

  const handleSubmit = () => {
    if (!isValid) {
      setError('Channel name must be lowercase letters, numbers, and hyphens only');
      return;
    }

    createChannel.mutate(
      { name, type, description: description || undefined },
      {
        onSuccess: (data) => {
          navigation.replace('Channel', {
            workspaceId,
            channelId: data.channel.id,
            channelName: data.channel.name,
          });
        },
        onError: (err) => {
          setError(err instanceof Error ? err.message : 'Failed to create channel');
        },
      },
    );
  };

  return (
    <View className="flex-1 bg-white px-4 pt-6 dark:bg-neutral-900">
      {/* Channel type */}
      <Text className="mb-2 text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
        Type
      </Text>
      <View className="mb-6 flex-row">
        <Pressable
          className={`mr-2 rounded-full px-4 py-2 ${type === 'public' ? 'bg-blue-500' : 'bg-neutral-200 dark:bg-neutral-700'}`}
          onPress={() => setType('public')}
        >
          <Text
            className={`text-sm font-medium ${type === 'public' ? 'text-white' : 'text-neutral-700 dark:text-neutral-300'}`}
          >
            # Public
          </Text>
        </Pressable>
        <Pressable
          className={`rounded-full px-4 py-2 ${type === 'private' ? 'bg-blue-500' : 'bg-neutral-200 dark:bg-neutral-700'}`}
          onPress={() => setType('private')}
        >
          <Text
            className={`text-sm font-medium ${type === 'private' ? 'text-white' : 'text-neutral-700 dark:text-neutral-300'}`}
          >
            🔒 Private
          </Text>
        </Pressable>
      </View>

      {/* Channel name */}
      <Text className="mb-2 text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
        Name
      </Text>
      <TextInput
        className="mb-1 rounded-lg border border-neutral-300 px-3 py-2.5 text-base text-neutral-900 dark:border-neutral-600 dark:text-white"
        placeholder="e.g. marketing"
        placeholderTextColor={isDark ? '#737373' : '#a3a3a3'}
        value={name}
        onChangeText={handleNameChange}
        autoCapitalize="none"
        autoCorrect={false}
        maxLength={80}
      />
      {error ? (
        <Text className="mb-4 text-sm text-red-500">{error}</Text>
      ) : (
        <Text className="mb-4 text-xs text-neutral-400 dark:text-neutral-500">
          Lowercase letters, numbers, and hyphens
        </Text>
      )}

      {/* Description */}
      <Text className="mb-2 text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
        Description (optional)
      </Text>
      <TextInput
        className="mb-6 rounded-lg border border-neutral-300 px-3 py-2.5 text-base text-neutral-900 dark:border-neutral-600 dark:text-white"
        placeholder="What's this channel about?"
        placeholderTextColor={isDark ? '#737373' : '#a3a3a3'}
        value={description}
        onChangeText={setDescription}
        multiline
        numberOfLines={3}
        textAlignVertical="top"
        maxLength={1000}
      />

      {/* Submit */}
      <Pressable
        className={`items-center rounded-lg py-3 ${isValid && !createChannel.isPending ? 'bg-blue-500 active:bg-blue-600' : 'bg-neutral-300 dark:bg-neutral-700'}`}
        onPress={handleSubmit}
        disabled={!isValid || createChannel.isPending}
      >
        {createChannel.isPending ? (
          <ActivityIndicator color="white" />
        ) : (
          <Text className="text-base font-semibold text-white">Create Channel</Text>
        )}
      </Pressable>
    </View>
  );
}
