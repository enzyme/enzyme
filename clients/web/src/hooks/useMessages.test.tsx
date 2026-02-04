import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { createMockMessageWithUser } from '../test-utils';

// Hoist mocks
const mockMessagesApi = vi.hoisted(() => ({
  get: vi.fn(),
  send: vi.fn(),
  list: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  addReaction: vi.fn(),
  removeReaction: vi.fn(),
  listThread: vi.fn(),
  markUnread: vi.fn(),
}));

vi.mock('../api/messages', () => ({
  messagesApi: mockMessagesApi,
}));

import {
  useMessages,
  useMessage,
  useSendMessage,
  useUpdateMessage,
  useDeleteMessage,
  useAddReaction,
  useRemoveReaction,
  useThreadMessages,
} from './useMessages';

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

function createWrapper(queryClient?: QueryClient) {
  const client = queryClient ?? createTestQueryClient();
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe('useMessages', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns loading state initially', () => {
    mockMessagesApi.list.mockImplementation(() => new Promise(() => {}));

    const { result } = renderHook(() => useMessages('channel-1'), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it('fetches messages for a channel', async () => {
    const messages = [
      createMockMessageWithUser({ id: 'msg-1', content: 'Hello' }),
      createMockMessageWithUser({ id: 'msg-2', content: 'World' }),
    ];
    mockMessagesApi.list.mockResolvedValue({
      messages,
      has_more: false,
      next_cursor: undefined,
    });

    const { result } = renderHook(() => useMessages('channel-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.pages[0].messages).toEqual(messages);
    expect(mockMessagesApi.list).toHaveBeenCalledWith('channel-1', {
      cursor: undefined,
      limit: 50,
      direction: 'before',
    });
  });

  it('does not fetch when channelId is undefined', () => {
    const { result } = renderHook(() => useMessages(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockMessagesApi.list).not.toHaveBeenCalled();
  });

  it('reports hasNextPage when more data is available', async () => {
    mockMessagesApi.list.mockResolvedValue({
      messages: [createMockMessageWithUser({ id: 'msg-1' })],
      has_more: true,
      next_cursor: 'cursor-1',
    });

    const { result } = renderHook(() => useMessages('channel-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.hasNextPage).toBe(true);
  });
});

describe('useMessage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches a single message', async () => {
    const message = createMockMessageWithUser({ id: 'msg-1', content: 'Test' });
    mockMessagesApi.get.mockResolvedValue({ message });

    const { result } = renderHook(() => useMessage('msg-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.message).toEqual(message);
  });

  it('does not fetch when messageId is undefined', () => {
    const { result } = renderHook(() => useMessage(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockMessagesApi.get).not.toHaveBeenCalled();
  });
});

describe('useSendMessage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('sends a message and updates cache', async () => {
    const queryClient = createTestQueryClient();
    const newMessage = createMockMessageWithUser({ id: 'msg-new', content: 'New message' });
    mockMessagesApi.send.mockResolvedValue({ message: newMessage });

    // Pre-populate cache with existing messages
    queryClient.setQueryData(['messages', 'channel-1'], {
      pages: [{ messages: [createMockMessageWithUser({ id: 'msg-1' })], has_more: false }],
      pageParams: [undefined],
    });

    const { result } = renderHook(() => useSendMessage('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ content: 'New message' });
    });

    expect(mockMessagesApi.send).toHaveBeenCalledWith('channel-1', { content: 'New message' });

    // Check cache was updated
    const cachedData = queryClient.getQueryData<{ pages: Array<{ messages: Array<{ id: string }> }> }>(['messages', 'channel-1']);
    expect(cachedData?.pages[0].messages[0].id).toBe('msg-new');
  });

  it('calls send API with correct parameters', async () => {
    const queryClient = createTestQueryClient();
    const newMessage = createMockMessageWithUser({ id: 'msg-new' });
    mockMessagesApi.send.mockResolvedValue({ message: newMessage });

    const { result } = renderHook(() => useSendMessage('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        content: 'Hello world',
        attachment_ids: ['file-1'],
      });
    });

    expect(mockMessagesApi.send).toHaveBeenCalledWith('channel-1', {
      content: 'Hello world',
      attachment_ids: ['file-1'],
    });
  });
});

describe('useUpdateMessage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls update API with correct parameters', async () => {
    const queryClient = createTestQueryClient();
    const updatedMessage = createMockMessageWithUser({ id: 'msg-1', content: 'Updated' });
    mockMessagesApi.update.mockResolvedValue({ message: updatedMessage });

    const { result } = renderHook(() => useUpdateMessage(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ messageId: 'msg-1', content: 'Updated' });
    });

    expect(mockMessagesApi.update).toHaveBeenCalledWith('msg-1', 'Updated');
  });

  it('returns updated message on success', async () => {
    const queryClient = createTestQueryClient();
    const updatedMessage = createMockMessageWithUser({ id: 'msg-1', content: 'Updated content' });
    mockMessagesApi.update.mockResolvedValue({ message: updatedMessage });

    const { result } = renderHook(() => useUpdateMessage(), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync({ messageId: 'msg-1', content: 'Updated content' });
    });

    expect(response).toEqual({ message: updatedMessage });
  });
});

describe('useDeleteMessage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls delete API with message ID', async () => {
    const queryClient = createTestQueryClient();
    mockMessagesApi.delete.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useDeleteMessage(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('msg-123');
    });

    expect(mockMessagesApi.delete).toHaveBeenCalledWith('msg-123');
  });

  it('returns success on delete', async () => {
    const queryClient = createTestQueryClient();
    mockMessagesApi.delete.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useDeleteMessage(), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync('msg-1');
    });

    expect(response).toEqual({ success: true });
  });
});

describe('useAddReaction', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls addReaction API with correct parameters', async () => {
    const queryClient = createTestQueryClient();
    mockMessagesApi.addReaction.mockResolvedValue({
      reaction: { id: 'r-1', message_id: 'msg-1', user_id: 'user-1', emoji: '👍', created_at: new Date().toISOString() },
    });

    const { result } = renderHook(() => useAddReaction('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ messageId: 'msg-1', emoji: '👍' });
    });

    expect(mockMessagesApi.addReaction).toHaveBeenCalledWith('msg-1', '👍');
  });

  it('returns reaction on success', async () => {
    const queryClient = createTestQueryClient();
    const reaction = { id: 'r-1', message_id: 'msg-1', user_id: 'user-1', emoji: '🔥', created_at: new Date().toISOString() };
    mockMessagesApi.addReaction.mockResolvedValue({ reaction });

    const { result } = renderHook(() => useAddReaction('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync({ messageId: 'msg-1', emoji: '🔥' });
    });

    expect(response).toEqual({ reaction });
  });
});

describe('useRemoveReaction', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls removeReaction API with correct parameters', async () => {
    const queryClient = createTestQueryClient();
    mockMessagesApi.removeReaction.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useRemoveReaction('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ messageId: 'msg-1', emoji: '👍' });
    });

    expect(mockMessagesApi.removeReaction).toHaveBeenCalledWith('msg-1', '👍');
  });

  it('returns success on removal', async () => {
    const queryClient = createTestQueryClient();
    mockMessagesApi.removeReaction.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useRemoveReaction('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync({ messageId: 'msg-1', emoji: '❤️' });
    });

    expect(response).toEqual({ success: true });
  });
});

describe('useThreadMessages', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches thread messages', async () => {
    const messages = [
      createMockMessageWithUser({ id: 'reply-1', content: 'Reply 1' }),
      createMockMessageWithUser({ id: 'reply-2', content: 'Reply 2' }),
    ];
    mockMessagesApi.listThread.mockResolvedValue({
      messages,
      has_more: false,
      next_cursor: undefined,
    });

    const { result } = renderHook(() => useThreadMessages('parent-msg-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.pages[0].messages).toEqual(messages);
    expect(mockMessagesApi.listThread).toHaveBeenCalledWith('parent-msg-1', {
      cursor: undefined,
      limit: 50,
    });
  });

  it('does not fetch when parentMessageId is undefined', () => {
    const { result } = renderHook(() => useThreadMessages(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockMessagesApi.listThread).not.toHaveBeenCalled();
  });
});
