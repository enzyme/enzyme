import { get, post, del, uploadFile, type User } from '@feather/api-client';

export interface UserProfile {
  id: string;
  display_name: string;
  avatar_url?: string;
  status: string;
  created_at: string;
}

export interface UpdateProfileInput {
  display_name?: string;
}

export interface AvatarUploadResponse {
  avatar_url: string;
}

export const usersApi = {
  getUser: (userId: string) => get<{ user: UserProfile }>(`/users/${userId}`),

  updateProfile: (input: UpdateProfileInput) => post<{ user: User }>('/users/me/profile', input),

  uploadAvatar: (file: File) =>
    uploadFile('/users/me/avatar', file) as Promise<AvatarUploadResponse>,

  deleteAvatar: () => del<{ success: boolean }>('/users/me/avatar'),
};
