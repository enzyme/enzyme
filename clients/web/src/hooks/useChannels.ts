import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { channelsApi, type CreateChannelInput, type CreateDMInput } from '../api/channels';

export function useChannels(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['channels', workspaceId],
    queryFn: () => channelsApi.list(workspaceId!),
    enabled: !!workspaceId,
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
