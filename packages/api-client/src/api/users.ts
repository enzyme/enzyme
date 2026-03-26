import { apiClient, throwIfError, multipartRequest } from '../client';
import type { UpdateProfileInput } from '../types';

export const usersApi = {
  getUser: (userId: string) =>
    throwIfError(apiClient.GET('/users/{id}', { params: { path: { id: userId } } })),

  updateProfile: (input: UpdateProfileInput) =>
    throwIfError(apiClient.POST('/users/me/profile', { body: input })),

  uploadAvatar: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return throwIfError(apiClient.POST('/users/me/avatar', multipartRequest(formData)));
  },

  deleteAvatar: () => throwIfError(apiClient.DELETE('/users/me/avatar')),
};
