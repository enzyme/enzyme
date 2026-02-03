import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { channelsApi } from '../api/channels';
import type { NotificationPreferences } from '@feather/api-client';

/**
 * Hook to get the current user's notification preferences for a channel
 */
export function useChannelNotifications(channelId: string | undefined) {
  return useQuery({
    queryKey: ['channel-notifications', channelId],
    queryFn: () => channelsApi.getNotifications(channelId!),
    enabled: !!channelId,
    staleTime: 30000, // 30 seconds
  });
}

/**
 * Hook to update notification preferences for a channel
 */
export function useUpdateChannelNotifications(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (preferences: NotificationPreferences) =>
      channelsApi.updateNotifications(channelId, preferences),
    onMutate: async (newPreferences) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['channel-notifications', channelId] });

      // Snapshot the previous value
      const previousData = queryClient.getQueryData<{ preferences: NotificationPreferences }>([
        'channel-notifications',
        channelId,
      ]);

      // Optimistically update
      queryClient.setQueryData(['channel-notifications', channelId], {
        preferences: newPreferences,
      });

      return { previousData };
    },
    onError: (_err, _newPreferences, context) => {
      // Rollback on error
      if (context?.previousData) {
        queryClient.setQueryData(['channel-notifications', channelId], context.previousData);
      }
    },
    onSettled: () => {
      // Invalidate to refetch
      queryClient.invalidateQueries({ queryKey: ['channel-notifications', channelId] });
    },
  });
}
