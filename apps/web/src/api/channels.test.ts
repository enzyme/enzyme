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

import { channelsApi } from './channels';
import { mockResponse } from '../test-utils/mocks/api-client';

describe('channelsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('create', () => {
    it('POST with name and type', async () => {
      const channel = { id: 'ch-1', name: 'general', type: 'public' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ channel }));

      const result = await channelsApi.create('ws-1', { name: 'general', type: 'public' });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/workspaces/{wid}/channels/create', {
        params: { path: { wid: 'ws-1' } },
        body: { name: 'general', type: 'public' },
      });
      expect(result).toEqual({ channel });
    });

    it('POST with optional description', async () => {
      const channel = { id: 'ch-1', name: 'help', type: 'public', description: 'Get help here' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ channel }));

      await channelsApi.create('ws-1', {
        name: 'help',
        type: 'public',
        description: 'Get help here',
      });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/workspaces/{wid}/channels/create', {
        params: { path: { wid: 'ws-1' } },
        body: { name: 'help', type: 'public', description: 'Get help here' },
      });
    });
  });

  describe('list', () => {
    it('POST with workspaceId', async () => {
      const channels = [
        { id: 'ch-1', name: 'general' },
        { id: 'ch-2', name: 'random' },
      ];
      mockApiClient.POST.mockResolvedValue(mockResponse({ channels }));

      const result = await channelsApi.list('ws-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/workspaces/{wid}/channels/list', {
        params: { path: { wid: 'ws-1' } },
      });
      expect(result).toEqual({ channels });
    });
  });

  describe('createDM', () => {
    it('POST DM with user_ids', async () => {
      const channel = { id: 'dm-1', type: 'dm' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ channel }));

      const result = await channelsApi.createDM('ws-1', { user_ids: ['user-2', 'user-3'] });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/workspaces/{wid}/channels/dm', {
        params: { path: { wid: 'ws-1' } },
        body: { user_ids: ['user-2', 'user-3'] },
      });
      expect(result).toEqual({ channel });
    });
  });

  describe('update', () => {
    it('POST with channelId and updates', async () => {
      const channel = { id: 'ch-1', name: 'updated-name' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ channel }));

      const result = await channelsApi.update('ch-1', { name: 'updated-name' });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/update', {
        params: { path: { id: 'ch-1' } },
        body: { name: 'updated-name' },
      });
      expect(result).toEqual({ channel });
    });

    it('POST with description', async () => {
      const channel = { id: 'ch-1', description: 'New description' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ channel }));

      await channelsApi.update('ch-1', { description: 'New description' });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/update', {
        params: { path: { id: 'ch-1' } },
        body: { description: 'New description' },
      });
    });
  });

  describe('archive', () => {
    it('POST archive', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.archive('ch-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/archive', {
        params: { path: { id: 'ch-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('addMember', () => {
    it('POST member with userId', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.addMember('ch-1', 'user-2');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/members/add', {
        params: { path: { id: 'ch-1' } },
        body: { user_id: 'user-2', role: undefined },
      });
      expect(result).toEqual({ success: true });
    });

    it('POST member with role', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      await channelsApi.addMember('ch-1', 'user-2', 'admin');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/members/add', {
        params: { path: { id: 'ch-1' } },
        body: { user_id: 'user-2', role: 'admin' },
      });
    });
  });

  describe('listMembers', () => {
    it('POST list members', async () => {
      const members = [
        { user_id: 'user-1', role: 'owner' },
        { user_id: 'user-2', role: 'member' },
      ];
      mockApiClient.POST.mockResolvedValue(mockResponse({ members }));

      const result = await channelsApi.listMembers('ch-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/members/list', {
        params: { path: { id: 'ch-1' } },
      });
      expect(result).toEqual({ members });
    });
  });

  describe('join', () => {
    it('POST join channel', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.join('ch-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/join', {
        params: { path: { id: 'ch-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('leave', () => {
    it('POST leave channel', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.leave('ch-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/leave', {
        params: { path: { id: 'ch-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('markAsRead', () => {
    it('POST read status without messageId', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.markAsRead('ch-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/mark-read', {
        params: { path: { id: 'ch-1' } },
        body: {},
      });
      expect(result).toEqual({ success: true });
    });

    it('POST read status with messageId', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      await channelsApi.markAsRead('ch-1', 'msg-5');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/mark-read', {
        params: { path: { id: 'ch-1' } },
        body: { message_id: 'msg-5' },
      });
    });
  });

  describe('markAllAsRead', () => {
    it('POST mark all as read in workspace', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.markAllAsRead('ws-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/workspaces/{wid}/channels/mark-all-read', {
        params: { path: { wid: 'ws-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('getNotifications', () => {
    it('GET notification preferences', async () => {
      const preferences = { muted: false, level: 'all' };
      mockApiClient.GET.mockResolvedValue(mockResponse({ preferences }));

      const result = await channelsApi.getNotifications('ch-1');

      expect(mockApiClient.GET).toHaveBeenCalledWith('/channels/{id}/notifications', {
        params: { path: { id: 'ch-1' } },
      });
      expect(result).toEqual({ preferences });
    });
  });

  describe('updateNotifications', () => {
    it('POST update notification preferences', async () => {
      const preferences = { notify_level: 'mentions' as const, email_enabled: false };
      mockApiClient.POST.mockResolvedValue(mockResponse({ preferences }));

      const result = await channelsApi.updateNotifications('ch-1', preferences);

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/notifications', {
        params: { path: { id: 'ch-1' } },
        body: preferences,
      });
      expect(result).toEqual({ preferences });
    });
  });

  describe('star', () => {
    it('POST star channel', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.star('ch-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/star', {
        params: { path: { id: 'ch-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('unstar', () => {
    it('DELETE unstar channel', async () => {
      mockApiClient.DELETE.mockResolvedValue(mockResponse({ success: true }));

      const result = await channelsApi.unstar('ch-1');

      expect(mockApiClient.DELETE).toHaveBeenCalledWith('/channels/{id}/star', {
        params: { path: { id: 'ch-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });
});
