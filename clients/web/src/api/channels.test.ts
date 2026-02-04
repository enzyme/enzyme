import { describe, it, expect, vi, beforeEach } from 'vitest';

// Hoist mocks
const mockGet = vi.hoisted(() => vi.fn());
const mockPost = vi.hoisted(() => vi.fn());
const mockDel = vi.hoisted(() => vi.fn());

vi.mock('@feather/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@feather/api-client')>();
  return {
    ...original,
    get: mockGet,
    post: mockPost,
    del: mockDel,
  };
});

import { channelsApi } from './channels';

describe('channelsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('create', () => {
    it('POST with name and type', async () => {
      const channel = { id: 'ch-1', name: 'general', type: 'public' };
      mockPost.mockResolvedValue({ channel });

      const result = await channelsApi.create('ws-1', { name: 'general', type: 'public' });

      expect(mockPost).toHaveBeenCalledWith('/workspaces/ws-1/channels/create', {
        name: 'general',
        type: 'public',
      });
      expect(result).toEqual({ channel });
    });

    it('POST with optional description', async () => {
      const channel = { id: 'ch-1', name: 'help', type: 'public', description: 'Get help here' };
      mockPost.mockResolvedValue({ channel });

      await channelsApi.create('ws-1', {
        name: 'help',
        type: 'public',
        description: 'Get help here',
      });

      expect(mockPost).toHaveBeenCalledWith('/workspaces/ws-1/channels/create', {
        name: 'help',
        type: 'public',
        description: 'Get help here',
      });
    });
  });

  describe('list', () => {
    it('GET with workspaceId', async () => {
      const channels = [
        { id: 'ch-1', name: 'general' },
        { id: 'ch-2', name: 'random' },
      ];
      mockPost.mockResolvedValue({ channels });

      const result = await channelsApi.list('ws-1');

      expect(mockPost).toHaveBeenCalledWith('/workspaces/ws-1/channels/list');
      expect(result).toEqual({ channels });
    });
  });

  describe('createDM', () => {
    it('POST DM with user_ids', async () => {
      const channel = { id: 'dm-1', type: 'dm' };
      mockPost.mockResolvedValue({ channel });

      const result = await channelsApi.createDM('ws-1', { user_ids: ['user-2', 'user-3'] });

      expect(mockPost).toHaveBeenCalledWith('/workspaces/ws-1/channels/dm', {
        user_ids: ['user-2', 'user-3'],
      });
      expect(result).toEqual({ channel });
    });
  });

  describe('update', () => {
    it('PATCH with channelId and updates', async () => {
      const channel = { id: 'ch-1', name: 'updated-name' };
      mockPost.mockResolvedValue({ channel });

      const result = await channelsApi.update('ch-1', { name: 'updated-name' });

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/update', {
        name: 'updated-name',
      });
      expect(result).toEqual({ channel });
    });

    it('PATCH with description', async () => {
      const channel = { id: 'ch-1', description: 'New description' };
      mockPost.mockResolvedValue({ channel });

      await channelsApi.update('ch-1', { description: 'New description' });

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/update', {
        description: 'New description',
      });
    });
  });

  describe('archive', () => {
    it('POST archive', async () => {
      mockPost.mockResolvedValue({ success: true });

      const result = await channelsApi.archive('ch-1');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/archive');
      expect(result).toEqual({ success: true });
    });
  });

  describe('addMember', () => {
    it('POST member with userId', async () => {
      mockPost.mockResolvedValue({ success: true });

      const result = await channelsApi.addMember('ch-1', 'user-2');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/members/add', {
        user_id: 'user-2',
        role: undefined,
      });
      expect(result).toEqual({ success: true });
    });

    it('POST member with role', async () => {
      mockPost.mockResolvedValue({ success: true });

      await channelsApi.addMember('ch-1', 'user-2', 'admin');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/members/add', {
        user_id: 'user-2',
        role: 'admin',
      });
    });
  });

  describe('listMembers', () => {
    it('POST list members', async () => {
      const members = [
        { user_id: 'user-1', role: 'owner' },
        { user_id: 'user-2', role: 'member' },
      ];
      mockPost.mockResolvedValue({ members });

      const result = await channelsApi.listMembers('ch-1');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/members/list');
      expect(result).toEqual({ members });
    });
  });

  describe('join', () => {
    it('POST join channel', async () => {
      mockPost.mockResolvedValue({ success: true });

      const result = await channelsApi.join('ch-1');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/join');
      expect(result).toEqual({ success: true });
    });
  });

  describe('leave', () => {
    it('POST leave channel', async () => {
      mockPost.mockResolvedValue({ success: true });

      const result = await channelsApi.leave('ch-1');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/leave');
      expect(result).toEqual({ success: true });
    });
  });

  describe('markAsRead', () => {
    it('POST read status without messageId', async () => {
      mockPost.mockResolvedValue({ success: true });

      const result = await channelsApi.markAsRead('ch-1');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/mark-read', {});
      expect(result).toEqual({ success: true });
    });

    it('POST read status with messageId', async () => {
      mockPost.mockResolvedValue({ success: true });

      await channelsApi.markAsRead('ch-1', 'msg-5');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/mark-read', {
        message_id: 'msg-5',
      });
    });
  });

  describe('markAllAsRead', () => {
    it('POST mark all as read in workspace', async () => {
      mockPost.mockResolvedValue({ success: true });

      const result = await channelsApi.markAllAsRead('ws-1');

      expect(mockPost).toHaveBeenCalledWith('/workspaces/ws-1/channels/mark-all-read');
      expect(result).toEqual({ success: true });
    });
  });

  describe('getNotifications', () => {
    it('GET notification preferences', async () => {
      const preferences = { muted: false, level: 'all' };
      mockGet.mockResolvedValue({ preferences });

      const result = await channelsApi.getNotifications('ch-1');

      expect(mockGet).toHaveBeenCalledWith('/channels/ch-1/notifications');
      expect(result).toEqual({ preferences });
    });
  });

  describe('updateNotifications', () => {
    it('POST update notification preferences', async () => {
      const preferences = { notify_level: 'mentions' as const, email_enabled: false };
      mockPost.mockResolvedValue({ preferences });

      const result = await channelsApi.updateNotifications('ch-1', preferences);

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/notifications', preferences);
      expect(result).toEqual({ preferences });
    });
  });

  describe('star', () => {
    it('POST star channel', async () => {
      mockPost.mockResolvedValue({ success: true });

      const result = await channelsApi.star('ch-1');

      expect(mockPost).toHaveBeenCalledWith('/channels/ch-1/star');
      expect(result).toEqual({ success: true });
    });
  });

  describe('unstar', () => {
    it('DELETE unstar channel', async () => {
      mockDel.mockResolvedValue({ success: true });

      const result = await channelsApi.unstar('ch-1');

      expect(mockDel).toHaveBeenCalledWith('/channels/ch-1/star');
      expect(result).toEqual({ success: true });
    });
  });
});
