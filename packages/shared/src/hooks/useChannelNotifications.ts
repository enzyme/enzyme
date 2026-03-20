import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { channelsApi, type NotificationPreferences } from '@enzyme/api-client';
import { channelKeys } from '../queryKeys';

/**
 * Hook to get the current user's notification preferences for a channel
 */
export function useChannelNotifications(channelId: string | undefined) {
  return useQuery({
    queryKey: channelKeys.notifications(channelId!),
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
      await queryClient.cancelQueries({ queryKey: channelKeys.notifications(channelId) });

      // Snapshot the previous value
      const previousData = queryClient.getQueryData<{ preferences: NotificationPreferences }>(
        channelKeys.notifications(channelId),
      );

      // Optimistically update
      queryClient.setQueryData(channelKeys.notifications(channelId), {
        preferences: newPreferences,
      });

      return { previousData };
    },
    onError: (_err, _newPreferences, context) => {
      // Rollback on error
      if (context?.previousData) {
        queryClient.setQueryData(channelKeys.notifications(channelId), context.previousData);
      }
    },
    onSettled: () => {
      // Invalidate to refetch
      queryClient.invalidateQueries({ queryKey: channelKeys.notifications(channelId) });
    },
  });
}
