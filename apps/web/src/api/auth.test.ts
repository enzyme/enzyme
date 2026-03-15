import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockApiClient = vi.hoisted(() => ({
  GET: vi.fn(),
  POST: vi.fn(),
}));

vi.mock('@enzyme/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@enzyme/api-client')>();
  return { ...original, apiClient: mockApiClient };
});

import { authApi } from './auth';
import { mockResponse } from '../test-utils/mocks/api-client';

describe('authApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('login', () => {
    it('POST /auth/login with credentials', async () => {
      const user = { id: 'user-1', email: 'test@example.com', display_name: 'Test' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ user }));

      const result = await authApi.login({ email: 'test@example.com', password: 'password123' });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/auth/login', {
        body: { email: 'test@example.com', password: 'password123' },
      });
      expect(result).toEqual({ user });
    });
  });

  describe('register', () => {
    it('POST /auth/register with user data', async () => {
      const user = { id: 'user-1', email: 'new@example.com', display_name: 'New User' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ user }));

      const result = await authApi.register({
        email: 'new@example.com',
        password: 'securepass',
        display_name: 'New User',
      });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/auth/register', {
        body: { email: 'new@example.com', password: 'securepass', display_name: 'New User' },
      });
      expect(result).toEqual({ user });
    });
  });

  describe('logout', () => {
    it('POST /auth/logout', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await authApi.logout();

      expect(mockApiClient.POST).toHaveBeenCalledWith('/auth/logout');
      expect(result).toEqual({ success: true });
    });
  });

  describe('me', () => {
    it('GET /auth/me', async () => {
      const user = { id: 'user-1', email: 'test@example.com', display_name: 'Test' };
      const workspaces = [{ id: 'ws-1', slug: 'test', name: 'Test Workspace', role: 'member' }];
      mockApiClient.GET.mockResolvedValue(mockResponse({ user, workspaces }));

      const result = await authApi.me();

      expect(mockApiClient.GET).toHaveBeenCalledWith('/auth/me');
      expect(result).toEqual({ user, workspaces });
    });

    it('returns user without workspaces when none exist', async () => {
      const user = { id: 'user-1', email: 'test@example.com', display_name: 'Test' };
      mockApiClient.GET.mockResolvedValue(mockResponse({ user }));

      const result = await authApi.me();

      expect(result).toEqual({ user });
    });
  });

  describe('forgotPassword', () => {
    it('POST /auth/forgot-password with email', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true, message: 'Reset email sent' }));

      const result = await authApi.forgotPassword('user@example.com');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/auth/forgot-password', {
        body: { email: 'user@example.com' },
      });
      expect(result).toEqual({ success: true, message: 'Reset email sent' });
    });
  });

  describe('resetPassword', () => {
    it('POST /auth/reset-password with token and new password', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await authApi.resetPassword('reset-token-123', 'newpassword456');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/auth/reset-password', {
        body: { token: 'reset-token-123', new_password: 'newpassword456' },
      });
      expect(result).toEqual({ success: true });
    });
  });
});
