import { useQuery, useMutation } from '@tanstack/react-query';
import { apiClient, throwIfError } from '@enzyme/api-client';
import type { VoiceParticipant } from '@enzyme/api-client';
import { voiceKeys } from '../queryKeys';

export function useVoiceParticipants(channelId: string | undefined) {
  return useQuery({
    queryKey: voiceKeys.participants(channelId!),
    queryFn: async (): Promise<VoiceParticipant[]> => {
      const result = await throwIfError(
        apiClient.GET('/channels/{id}/voice/participants', {
          params: { path: { id: channelId! } },
        }),
      );
      return result.participants;
    },
    enabled: !!channelId,
    staleTime: 10000,
  });
}

export function useServerMuteVoice() {
  return useMutation({
    mutationFn: async ({
      channelId,
      userId,
      muted,
    }: {
      channelId: string;
      userId: string;
      muted: boolean;
    }) => {
      return throwIfError(
        apiClient.POST('/channels/{id}/voice/server-mute', {
          params: { path: { id: channelId } },
          body: { user_id: userId, muted },
        }),
      );
    },
  });
}
