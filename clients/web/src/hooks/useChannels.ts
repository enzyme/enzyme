import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  channelsApi,
  type CreateChannelInput,
  type CreateDMInput,
  type UpdateChannelInput,
  type ConvertGroupDMInput,
} from '../api/channels';
import type { ChannelWithMembership } from '@feather/api-client';

export function useChannels(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['channels', workspaceId],
    queryFn: () => channelsApi.list(workspaceId!),
    enabled: !!workspaceId,
  });
}

export function useMarkChannelAsRead(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ channelId, messageId }: { channelId: string; messageId?: string }) =>
      channelsApi.markAsRead(channelId, messageId),
    onSuccess: (data, { channelId }) => {
      // Update the channel's unread_count and last_read_message_id
      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) =>
              c.id === channelId
                ? { ...c, unread_count: 0, last_read_message_id: data.last_read_message_id }
                : c,
            ),
          };
        },
      );
    },
  });
}

export function useMarkAllChannelsAsRead(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => channelsApi.markAllAsRead(workspaceId),
    onSuccess: () => {
      // Set all unread_count to 0
      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) => ({ ...c, unread_count: 0 })),
          };
        },
      );
    },
  });
}

export function useChannelMembers(channelId: string | undefined) {
  return useQuery({
    queryKey: ['channel', channelId, 'members'],
    queryFn: () => channelsApi.listMembers(channelId!),
    enabled: !!channelId,
  });
}

export function useCreateChannel(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateChannelInput) => channelsApi.create(workspaceId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useCreateDM(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateDMInput) => channelsApi.createDM(workspaceId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useJoinChannel(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (channelId: string) => channelsApi.join(channelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useLeaveChannel(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (channelId: string) => channelsApi.leave(channelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useArchiveChannel(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (channelId: string) => channelsApi.archive(channelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useAddChannelMember(channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role?: string }) =>
      channelsApi.addMember(channelId, userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channel', channelId, 'members'] });
    },
  });
}

export function useUpdateChannel(workspaceId: string, channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: UpdateChannelInput) => channelsApi.update(channelId, input),
    onSuccess: (data) => {
      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) => (c.id === channelId ? { ...c, ...data.channel } : c)),
          };
        },
      );
    },
  });
}

export function useStarChannel(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (channelId: string) => channelsApi.star(channelId),
    onMutate: async (channelId) => {
      await queryClient.cancelQueries({ queryKey: ['channels', workspaceId] });

      const previousData = queryClient.getQueryData<{ channels: ChannelWithMembership[] }>([
        'channels',
        workspaceId,
      ]);

      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) =>
              c.id === channelId ? { ...c, is_starred: true } : c,
            ),
          };
        },
      );

      return { previousData };
    },
    onError: (_err, _channelId, context) => {
      if (context?.previousData) {
        queryClient.setQueryData(['channels', workspaceId], context.previousData);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useUnstarChannel(workspaceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (channelId: string) => channelsApi.unstar(channelId),
    onMutate: async (channelId) => {
      await queryClient.cancelQueries({ queryKey: ['channels', workspaceId] });

      const previousData = queryClient.getQueryData<{ channels: ChannelWithMembership[] }>([
        'channels',
        workspaceId,
      ]);

      queryClient.setQueryData(
        ['channels', workspaceId],
        (old: { channels: ChannelWithMembership[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            channels: old.channels.map((c) =>
              c.id === channelId ? { ...c, is_starred: false } : c,
            ),
          };
        },
      );

      return { previousData };
    },
    onError: (_err, _channelId, context) => {
      if (context?.previousData) {
        queryClient.setQueryData(['channels', workspaceId], context.previousData);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useConvertGroupDMToChannel(workspaceId: string, channelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: ConvertGroupDMInput) => channelsApi.convertGroupDM(channelId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] });
    },
  });
}

export function useDMSuggestions(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['dm-suggestions', workspaceId],
    queryFn: () => channelsApi.getDMSuggestions(workspaceId!),
    enabled: !!workspaceId,
    staleTime: 60000, // Cache for 1 minute
  });
}
