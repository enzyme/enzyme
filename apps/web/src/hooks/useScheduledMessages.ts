import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  scheduledMessagesApi,
  type ScheduleMessageInput,
  type UpdateScheduledMessageInput,
} from '@enzyme/api-client';
import { scheduledMessageKeys } from '@enzyme/shared';
import { toast } from '../components/ui';

export function useScheduledMessages(workspaceId: string) {
  return useQuery({
    queryKey: scheduledMessageKeys.list(workspaceId),
    queryFn: () => scheduledMessagesApi.list(workspaceId),
    enabled: !!workspaceId,
  });
}

export function useScheduleMessage(channelId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ScheduleMessageInput) => scheduledMessagesApi.schedule(channelId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduledMessageKeys.all });
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to schedule message', 'error');
    },
  });
}

export function useUpdateScheduledMessage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateScheduledMessageInput }) =>
      scheduledMessagesApi.update(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduledMessageKeys.all });
      toast('Scheduled message updated', 'success');
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to update scheduled message', 'error');
    },
  });
}

export function useDeleteScheduledMessage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => scheduledMessagesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduledMessageKeys.all });
      toast('Scheduled message canceled', 'success');
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to cancel scheduled message', 'error');
    },
  });
}

export function useSendScheduledMessageNow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => scheduledMessagesApi.sendNow(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduledMessageKeys.all });
      toast('Message sent', 'success');
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to send message', 'error');
    },
  });
}

export function useRetryScheduledMessage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => {
      const newTime = new Date(Date.now() + 60 * 1000).toISOString();
      return scheduledMessagesApi.update(id, { scheduled_for: newTime });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduledMessageKeys.all });
      toast('Message rescheduled', 'success');
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to retry scheduled message', 'error');
    },
  });
}
