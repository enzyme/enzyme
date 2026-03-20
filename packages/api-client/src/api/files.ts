import { apiClient, throwIfError, multipartRequest } from '../client';

export const filesApi = {
  upload: (channelId: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return throwIfError(
      apiClient.POST('/channels/{id}/files/upload', {
        params: { path: { id: channelId } },
        ...multipartRequest(formData),
      }),
    );
  },

  signUrl: (fileId: string) =>
    throwIfError(apiClient.POST('/files/{id}/sign-url', { params: { path: { id: fileId } } })),

  signUrls: (fileIds: string[]) =>
    throwIfError(apiClient.POST('/files/sign-urls', { body: { file_ids: fileIds } })),
};
