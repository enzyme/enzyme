import { View, Text, Pressable, ActivityIndicator } from 'react-native';
import type { MainScreenProps } from '../navigation/types';
import { useWorkspaceMembers, useCreateDM, useAuth } from '@enzyme/shared';
import { Avatar } from '../components/Avatar';

export function ProfileScreen({ route, navigation }: MainScreenProps<'Profile'>) {
  const { workspaceId, userId } = route.params;
  const { user: currentUser } = useAuth();
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const member = membersData?.members?.find((m) => m.user_id === userId);
  const createDM = useCreateDM(workspaceId);

  const isOwnProfile = currentUser?.id === userId;

  const handleSendMessage = async () => {
    const result = await createDM.mutateAsync({ user_ids: [userId] });
    if (result.channel) {
      navigation.navigate('Channel', {
        workspaceId,
        channelId: result.channel.id,
        channelName: member?.display_name ?? 'DM',
      });
    }
  };

  if (!member) {
    return (
      <View className="flex-1 items-center justify-center bg-white dark:bg-neutral-900">
        <ActivityIndicator size="large" />
      </View>
    );
  }

  return (
    <View className="flex-1 bg-white px-6 pt-8 dark:bg-neutral-900">
      <View className="items-center">
        <Avatar
          user={{
            display_name: member.display_name,
            avatar_url: member.avatar_url,
            gravatar_url: member.gravatar_url,
          }}
          size="lg"
        />
        <Text className="mt-4 text-xl font-bold text-neutral-900 dark:text-white">
          {member.display_name}
        </Text>
        <Text className="mt-1 text-base text-neutral-500 dark:text-neutral-400">
          {member.email}
        </Text>
        <Text className="mt-2 text-sm capitalize text-neutral-500 dark:text-neutral-400">
          {member.role}
        </Text>

        {!isOwnProfile && (
          <Pressable
            className="mt-6 rounded-lg bg-blue-500 px-6 py-2.5 active:bg-blue-600"
            onPress={handleSendMessage}
            disabled={createDM.isPending}
          >
            {createDM.isPending ? (
              <ActivityIndicator color="white" size="small" />
            ) : (
              <Text className="text-base font-semibold text-white">Send message</Text>
            )}
          </Pressable>
        )}
      </View>
    </View>
  );
}
