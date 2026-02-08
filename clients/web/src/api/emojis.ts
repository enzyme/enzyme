import { post, uploadFile } from '@feather/api-client';
import type { CustomEmoji } from '@feather/api-client';

export const emojisApi = {
  list: (workspaceId: string) =>
    post<{ emojis: CustomEmoji[] }>(`/workspaces/${workspaceId}/emojis/list`),

  upload: (workspaceId: string, file: File, name: string) =>
    uploadFile(`/workspaces/${workspaceId}/emojis/upload`, file, { name }) as Promise<{ emoji: CustomEmoji }>,

  delete: (emojiId: string) =>
    post<{ success: boolean }>(`/emojis/${emojiId}/delete`),
};
