import { describe, it, expect, vi, beforeEach } from 'vitest';

// Hoist mocks
const mockGet = vi.hoisted(() => vi.fn());
const mockPost = vi.hoisted(() => vi.fn());
const mockDel = vi.hoisted(() => vi.fn());
const mockUploadFile = vi.hoisted(() => vi.fn());

vi.mock('@feather/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@feather/api-client')>();
  return {
    ...original,
    get: mockGet,
    post: mockPost,
    del: mockDel,
    uploadFile: mockUploadFile,
  };
});

import { usersApi } from './users';

describe('usersApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getUser', () => {
    it('GET /users/:id', async () => {
      const user = { id: 'user-1', display_name: 'Test User', status: 'active' };
      mockGet.mockResolvedValue({ user });

      const result = await usersApi.getUser('user-1');

      expect(mockGet).toHaveBeenCalledWith('/users/user-1');
      expect(result).toEqual({ user });
    });
  });

  describe('updateProfile', () => {
    it('PATCH /users/me/profile with display_name', async () => {
      const user = { id: 'user-1', display_name: 'Updated Name', email: 'test@example.com' };
      mockPost.mockResolvedValue({ user });

      const result = await usersApi.updateProfile({ display_name: 'Updated Name' });

      expect(mockPost).toHaveBeenCalledWith('/users/me/profile', {
        display_name: 'Updated Name',
      });
      expect(result).toEqual({ user });
    });

    it('PATCH with empty update', async () => {
      const user = { id: 'user-1', display_name: 'Same Name' };
      mockPost.mockResolvedValue({ user });

      const result = await usersApi.updateProfile({});

      expect(mockPost).toHaveBeenCalledWith('/users/me/profile', {});
      expect(result).toEqual({ user });
    });
  });

  describe('uploadAvatar', () => {
    it('POST with FormData file', async () => {
      mockUploadFile.mockResolvedValue({ avatar_url: 'https://example.com/avatar.jpg' });

      const file = new File(['test image'], 'avatar.jpg', { type: 'image/jpeg' });
      const result = await usersApi.uploadAvatar(file);

      expect(mockUploadFile).toHaveBeenCalledWith('/users/me/avatar', file);
      expect(result).toEqual({ avatar_url: 'https://example.com/avatar.jpg' });
    });
  });

  describe('deleteAvatar', () => {
    it('DELETE /users/me/avatar', async () => {
      mockDel.mockResolvedValue({ success: true });

      const result = await usersApi.deleteAvatar();

      expect(mockDel).toHaveBeenCalledWith('/users/me/avatar');
      expect(result).toEqual({ success: true });
    });
  });
});
