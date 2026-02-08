import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { emojisApi } from '../api/emojis';
import { toast } from '../components/ui';
import type { CustomEmoji } from '@feather/api-client';

export function useCustomEmojis(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['custom-emojis', workspaceId],
    queryFn: () => emojisApi.list(workspaceId!),
    enabled: !!workspaceId,
    staleTime: 5 * 60 * 1000,
    select: (data) => data.emojis,
  });
}

export function useCustomEmojiMap(workspaceId: string | undefined) {
  const { data: emojis } = useCustomEmojis(workspaceId);
  return useMemo(() => {
    const map = new Map<string, CustomEmoji>();
    if (emojis) {
      for (const emoji of emojis) {
        map.set(emoji.name, emoji);
      }
    }
    return map;
  }, [emojis]);
}

export function useUploadCustomEmoji(workspaceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file, name }: { file: File; name: string }) =>
      emojisApi.upload(workspaceId, file, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['custom-emojis', workspaceId] });
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to upload emoji', 'error');
    },
  });
}

export function useDeleteCustomEmoji(workspaceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (emojiId: string) => emojisApi.delete(emojiId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['custom-emojis', workspaceId] });
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to delete emoji', 'error');
    },
  });
}
