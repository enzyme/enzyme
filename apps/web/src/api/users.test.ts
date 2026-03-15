import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockApiClient = vi.hoisted(() => ({
  GET: vi.fn(),
  POST: vi.fn(),
  DELETE: vi.fn(),
}));

vi.mock('@enzyme/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@enzyme/api-client')>();
  return { ...original, apiClient: mockApiClient };
});

import { usersApi } from './users';
import { mockResponse } from '../test-utils/mocks/api-client';

describe('usersApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getUser', () => {
    it('GET /users/:id', async () => {
      const user = { id: 'user-1', display_name: 'Test User', status: 'active' };
      mockApiClient.GET.mockResolvedValue(mockResponse({ user }));

      const result = await usersApi.getUser('user-1');

      expect(mockApiClient.GET).toHaveBeenCalledWith('/users/{id}', {
        params: { path: { id: 'user-1' } },
      });
      expect(result).toEqual({ user });
    });
  });

  describe('updateProfile', () => {
    it('POST /users/me/profile with display_name', async () => {
      const user = { id: 'user-1', display_name: 'Updated Name', email: 'test@example.com' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ user }));

      const result = await usersApi.updateProfile({ display_name: 'Updated Name' });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/users/me/profile', {
        body: { display_name: 'Updated Name' },
      });
      expect(result).toEqual({ user });
    });

    it('POST with empty update', async () => {
      const user = { id: 'user-1', display_name: 'Same Name' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ user }));

      const result = await usersApi.updateProfile({});

      expect(mockApiClient.POST).toHaveBeenCalledWith('/users/me/profile', { body: {} });
      expect(result).toEqual({ user });
    });
  });

  describe('uploadAvatar', () => {
    it('POST with FormData file', async () => {
      mockApiClient.POST.mockResolvedValue(
        mockResponse({ avatar_url: 'https://example.com/avatar.jpg' }),
      );

      const file = new File(['test image'], 'avatar.jpg', { type: 'image/jpeg' });
      const result = await usersApi.uploadAvatar(file);

      expect(mockApiClient.POST).toHaveBeenCalledWith(
        '/users/me/avatar',
        expect.objectContaining({ bodySerializer: expect.any(Function) }),
      );
      expect(result).toEqual({ avatar_url: 'https://example.com/avatar.jpg' });
    });
  });

  describe('deleteAvatar', () => {
    it('DELETE /users/me/avatar', async () => {
      mockApiClient.DELETE.mockResolvedValue(mockResponse({ success: true }));

      const result = await usersApi.deleteAvatar();

      expect(mockApiClient.DELETE).toHaveBeenCalledWith('/users/me/avatar');
      expect(result).toEqual({ success: true });
    });
  });
});
