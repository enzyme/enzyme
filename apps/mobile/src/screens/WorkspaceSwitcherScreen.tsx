import { View, Text, FlatList, Pressable, ActivityIndicator, Alert } from 'react-native';
import { useAuth } from '@enzyme/shared';
import type { MainScreenProps } from '../navigation/types';
import type { WorkspaceSummary } from '@enzyme/api-client';
import { Avatar } from '../components/Avatar';
import { unregisterPushToken } from '../lib/notifications';

export function WorkspaceSwitcherScreen({ navigation }: MainScreenProps<'WorkspaceSwitcher'>) {
  const { user, workspaces, logout, isLoggingOut } = useAuth();

  const handleLogout = () => {
    Alert.alert('Sign out', 'Are you sure you want to sign out?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign out',
        style: 'destructive',
        onPress: async () => {
          // Best-effort unregistration with timeout; don't block logout indefinitely
          try {
            await Promise.race([
              unregisterPushToken(),
              new Promise<void>((resolve) => setTimeout(resolve, 3000)),
            ]);
          } catch {
            // Don't block logout on unregistration failure
          }
          logout();
        },
      },
    ]);
  };

  const renderWorkspace = ({ item }: { item: WorkspaceSummary }) => (
    <Pressable
      className="mx-4 mb-2 flex-row items-center rounded-xl bg-neutral-50 px-4 py-3 active:bg-neutral-100 dark:bg-neutral-800 dark:active:bg-neutral-700"
      onPress={() => navigation.navigate('ChannelList', { workspaceId: item.id })}
    >
      <Avatar
        user={{ display_name: item.name, id: item.id, avatar_url: item.icon_url }}
        size="md"
      />
      <View className="ml-3 flex-1">
        <Text className="text-base font-semibold text-neutral-900 dark:text-white">
          {item.name}
        </Text>
        <Text className="text-sm capitalize text-neutral-500 dark:text-neutral-400">
          {item.role}
        </Text>
      </View>
    </Pressable>
  );

  return (
    <View className="flex-1 bg-white dark:bg-neutral-900">
      {/* User header */}
      <View className="flex-row items-center justify-between border-b border-neutral-200 px-4 py-3 dark:border-neutral-700">
        <View>
          <Text className="text-base font-medium text-neutral-900 dark:text-white">
            {user?.display_name}
          </Text>
          <Text className="text-sm text-neutral-500 dark:text-neutral-400">{user?.email}</Text>
        </View>
        <Pressable
          className="rounded-lg px-3 py-1.5 active:bg-neutral-100 dark:active:bg-neutral-800"
          onPress={handleLogout}
          disabled={isLoggingOut}
        >
          {isLoggingOut ? (
            <ActivityIndicator size="small" />
          ) : (
            <Text className="text-sm font-medium text-red-500">Sign out</Text>
          )}
        </Pressable>
      </View>

      <FlatList
        data={workspaces}
        keyExtractor={(item) => item.id}
        renderItem={renderWorkspace}
        contentContainerStyle={{ paddingTop: 12 }}
        ListEmptyComponent={
          <View className="flex-1 items-center justify-center px-8 pt-20">
            <Text className="text-center text-base text-neutral-500 dark:text-neutral-400">
              You&apos;re not a member of any workspaces yet.
            </Text>
          </View>
        }
      />
    </View>
  );
}
