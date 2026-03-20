import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { messagesApi, type ThreadSubscriptionStatus } from '@enzyme/api-client';
import { threadKeys } from '../queryKeys';

/**
 * Hook to get the current user's subscription status for a thread
 */
export function useThreadSubscription(messageId: string | undefined) {
  return useQuery({
    queryKey: threadKeys.subscription(messageId!),
    queryFn: () => messagesApi.getThreadSubscription(messageId!),
    enabled: !!messageId,
    staleTime: 30000, // 30 seconds
  });
}

/**
 * Hook to subscribe to a thread
 */
export function useSubscribeToThread() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => messagesApi.subscribeToThread(messageId),
    onMutate: async (messageId) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: threadKeys.subscription(messageId) });

      // Snapshot the previous value
      const previousStatus = queryClient.getQueryData<{ status: ThreadSubscriptionStatus }>(
        threadKeys.subscription(messageId),
      );

      // Optimistically update to subscribed
      queryClient.setQueryData(threadKeys.subscription(messageId), {
        status: 'subscribed' as ThreadSubscriptionStatus,
      });

      return { previousStatus };
    },
    onError: (_err, messageId, context) => {
      // Rollback on error
      if (context?.previousStatus) {
        queryClient.setQueryData(threadKeys.subscription(messageId), context.previousStatus);
      }
    },
    onSettled: (_data, _err, messageId) => {
      // Invalidate to refetch
      queryClient.invalidateQueries({ queryKey: threadKeys.subscription(messageId) });
    },
  });
}

/**
 * Hook to unsubscribe from a thread
 */
export function useUnsubscribeFromThread() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (messageId: string) => messagesApi.unsubscribeFromThread(messageId),
    onMutate: async (messageId) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: threadKeys.subscription(messageId) });

      // Snapshot the previous value
      const previousStatus = queryClient.getQueryData<{ status: ThreadSubscriptionStatus }>(
        threadKeys.subscription(messageId),
      );

      // Optimistically update to unsubscribed
      queryClient.setQueryData(threadKeys.subscription(messageId), {
        status: 'unsubscribed' as ThreadSubscriptionStatus,
      });

      return { previousStatus };
    },
    onError: (_err, messageId, context) => {
      // Rollback on error
      if (context?.previousStatus) {
        queryClient.setQueryData(threadKeys.subscription(messageId), context.previousStatus);
      }
    },
    onSettled: (_data, _err, messageId) => {
      // Invalidate to refetch
      queryClient.invalidateQueries({ queryKey: threadKeys.subscription(messageId) });
    },
  });
}
