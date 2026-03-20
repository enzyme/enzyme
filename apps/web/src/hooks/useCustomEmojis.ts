import { useMutation, useQueryClient } from '@tanstack/react-query';
import { emojisApi } from '@enzyme/api-client';
import { emojiKeys } from '@enzyme/shared';
import { toast } from '../components/ui';

// Re-export the pure hooks from shared
export { useCustomEmojis, useCustomEmojiMap } from '@enzyme/shared';

// Web-specific: adds toast error handling
export function useUploadCustomEmoji(workspaceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file, name }: { file: File; name: string }) =>
      emojisApi.upload(workspaceId, file, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: emojiKeys.list(workspaceId) });
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to upload emoji', 'error');
    },
  });
}

// Web-specific: adds toast error handling
export function useDeleteCustomEmoji(workspaceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (emojiId: string) => emojisApi.delete(emojiId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: emojiKeys.list(workspaceId) });
    },
    onError: (error) => {
      toast(error instanceof Error ? error.message : 'Failed to delete emoji', 'error');
    },
  });
}
