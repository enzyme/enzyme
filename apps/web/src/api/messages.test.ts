import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockApiClient = vi.hoisted(() => ({
  GET: vi.fn(),
  POST: vi.fn(),
}));

vi.mock('@enzyme/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@enzyme/api-client')>();
  return { ...original, apiClient: mockApiClient };
});

import { messagesApi } from './messages';
import { mockResponse } from '../test-utils/mocks/api-client';

describe('messagesApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('get', () => {
    it('GET /messages/:id', async () => {
      const message = { id: 'msg-1', content: 'Hello', channel_id: 'ch-1' };
      mockApiClient.GET.mockResolvedValue(mockResponse({ message }));

      const result = await messagesApi.get('msg-1');

      expect(mockApiClient.GET).toHaveBeenCalledWith('/messages/{id}', {
        params: { path: { id: 'msg-1' } },
      });
      expect(result).toEqual({ message });
    });
  });

  describe('send', () => {
    it('POST with channelId and content', async () => {
      const message = { id: 'msg-new', content: 'New message', channel_id: 'ch-1' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ message }));

      const result = await messagesApi.send('ch-1', { content: 'New message' });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/messages/send', {
        params: { path: { id: 'ch-1' } },
        body: { content: 'New message' },
      });
      expect(result).toEqual({ message });
    });

    it('POST with thread_parent_id for threaded reply', async () => {
      const message = { id: 'msg-reply', content: 'Reply', thread_parent_id: 'msg-parent' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ message }));

      const result = await messagesApi.send('ch-1', {
        content: 'Reply',
        thread_parent_id: 'msg-parent',
      });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/messages/send', {
        params: { path: { id: 'ch-1' } },
        body: { content: 'Reply', thread_parent_id: 'msg-parent' },
      });
      expect(result).toEqual({ message });
    });

    it('POST with attachment_ids', async () => {
      const message = { id: 'msg-attach', content: 'With file', attachment_ids: ['file-1'] };
      mockApiClient.POST.mockResolvedValue(mockResponse({ message }));

      await messagesApi.send('ch-1', {
        content: 'With file',
        attachment_ids: ['file-1'],
      });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/messages/send', {
        params: { path: { id: 'ch-1' } },
        body: { content: 'With file', attachment_ids: ['file-1'] },
      });
    });
  });

  describe('list', () => {
    it('POST with channelId and pagination params', async () => {
      const messages = [{ id: 'msg-1' }, { id: 'msg-2' }];
      mockApiClient.POST.mockResolvedValue(mockResponse({ messages, has_more: false }));

      const result = await messagesApi.list('ch-1', {
        cursor: 'cursor-1',
        limit: 25,
        direction: 'before',
      });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/messages/list', {
        params: { path: { id: 'ch-1' } },
        body: { cursor: 'cursor-1', limit: 25, direction: 'before' },
      });
      expect(result).toEqual({ messages, has_more: false });
    });

    it('POST with empty input uses default', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ messages: [], has_more: false }));

      await messagesApi.list('ch-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/channels/{id}/messages/list', {
        params: { path: { id: 'ch-1' } },
        body: {},
      });
    });
  });

  describe('update', () => {
    it('POST with messageId and content', async () => {
      const message = { id: 'msg-1', content: 'Updated content' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ message }));

      const result = await messagesApi.update('msg-1', 'Updated content');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/update', {
        params: { path: { id: 'msg-1' } },
        body: { content: 'Updated content' },
      });
      expect(result).toEqual({ message });
    });
  });

  describe('delete', () => {
    it('POST with messageId', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await messagesApi.delete('msg-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/delete', {
        params: { path: { id: 'msg-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('addReaction', () => {
    it('POST reaction with emoji', async () => {
      const reaction = { id: 'r-1', message_id: 'msg-1', user_id: 'user-1', emoji: '👍' };
      mockApiClient.POST.mockResolvedValue(mockResponse({ reaction }));

      const result = await messagesApi.addReaction('msg-1', '👍');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/reactions/add', {
        params: { path: { id: 'msg-1' } },
        body: { emoji: '👍' },
      });
      expect(result).toEqual({ reaction });
    });
  });

  describe('removeReaction', () => {
    it('POST with emoji', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await messagesApi.removeReaction('msg-1', '👍');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/reactions/remove', {
        params: { path: { id: 'msg-1' } },
        body: { emoji: '👍' },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('listThread', () => {
    it('POST thread messages with pagination', async () => {
      const messages = [{ id: 'reply-1' }, { id: 'reply-2' }];
      mockApiClient.POST.mockResolvedValue(mockResponse({ messages, has_more: false }));

      const result = await messagesApi.listThread('msg-parent', { limit: 50 });

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/thread/list', {
        params: { path: { id: 'msg-parent' } },
        body: { limit: 50 },
      });
      expect(result).toEqual({ messages, has_more: false });
    });

    it('POST with empty input uses default', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ messages: [], has_more: false }));

      await messagesApi.listThread('msg-parent');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/thread/list', {
        params: { path: { id: 'msg-parent' } },
        body: {},
      });
    });
  });

  describe('markUnread', () => {
    it('POST mark-unread', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ success: true }));

      const result = await messagesApi.markUnread('msg-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/mark-unread', {
        params: { path: { id: 'msg-1' } },
      });
      expect(result).toEqual({ success: true });
    });
  });

  describe('getThreadSubscription', () => {
    it('GET subscription status', async () => {
      mockApiClient.GET.mockResolvedValue(mockResponse({ status: 'subscribed' }));

      const result = await messagesApi.getThreadSubscription('msg-1');

      expect(mockApiClient.GET).toHaveBeenCalledWith('/messages/{id}/subscription', {
        params: { path: { id: 'msg-1' } },
      });
      expect(result).toEqual({ status: 'subscribed' });
    });
  });

  describe('subscribeToThread', () => {
    it('POST subscribe', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ status: 'subscribed' }));

      const result = await messagesApi.subscribeToThread('msg-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/subscribe', {
        params: { path: { id: 'msg-1' } },
      });
      expect(result).toEqual({ status: 'subscribed' });
    });
  });

  describe('unsubscribeFromThread', () => {
    it('POST unsubscribe', async () => {
      mockApiClient.POST.mockResolvedValue(mockResponse({ status: 'unsubscribed' }));

      const result = await messagesApi.unsubscribeFromThread('msg-1');

      expect(mockApiClient.POST).toHaveBeenCalledWith('/messages/{id}/unsubscribe', {
        params: { path: { id: 'msg-1' } },
      });
      expect(result).toEqual({ status: 'unsubscribed' });
    });
  });
});
