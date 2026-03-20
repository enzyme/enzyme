import { useMutation } from '@tanstack/react-query';
import { filesApi } from '@enzyme/api-client';

export function useUploadFile(channelId: string) {
  return useMutation({
    mutationFn: (file: File) => filesApi.upload(channelId, file),
  });
}
