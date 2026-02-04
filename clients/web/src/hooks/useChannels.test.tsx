import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import type { ChannelWithMembership } from '@feather/api-client';

// Hoist mocks
const mockChannelsApi = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  createDM: vi.fn(),
  update: vi.fn(),
  archive: vi.fn(),
  join: vi.fn(),
  leave: vi.fn(),
  markAsRead: vi.fn(),
  markAllAsRead: vi.fn(),
  addMember: vi.fn(),
  listMembers: vi.fn(),
  star: vi.fn(),
  unstar: vi.fn(),
}));

vi.mock('../api/channels', () => ({
  channelsApi: mockChannelsApi,
}));

import {
  useChannels,
  useCreateChannel,
  useCreateDM,
  useJoinChannel,
  useLeaveChannel,
  useMarkChannelAsRead,
  useMarkAllChannelsAsRead,
  useStarChannel,
  useUnstarChannel,
  useUpdateChannel,
  useChannelMembers,
} from './useChannels';

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

function createMockChannel(overrides: Partial<ChannelWithMembership> = {}): ChannelWithMembership {
  return {
    id: overrides.id ?? 'ch-1',
    workspace_id: 'ws-1',
    name: 'general',
    type: 'public',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    is_member: true,
    is_starred: false,
    unread_count: 0,
    ...overrides,
  };
}

describe('useChannels', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns loading state initially', () => {
    mockChannelsApi.list.mockImplementation(() => new Promise(() => {}));

    const { result } = renderHook(() => useChannels('ws-1'), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it('fetches channels for a workspace', async () => {
    const channels = [
      createMockChannel({ id: 'ch-1', name: 'general' }),
      createMockChannel({ id: 'ch-2', name: 'random' }),
    ];
    mockChannelsApi.list.mockResolvedValue({ channels });

    const { result } = renderHook(() => useChannels('ws-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.channels).toEqual(channels);
    expect(mockChannelsApi.list).toHaveBeenCalledWith('ws-1');
  });

  it('does not fetch when workspaceId is undefined', () => {
    const { result } = renderHook(() => useChannels(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockChannelsApi.list).not.toHaveBeenCalled();
  });
});

describe('useCreateChannel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates a channel and invalidates cache', async () => {
    const queryClient = createTestQueryClient();
    const newChannel = createMockChannel({ id: 'ch-new', name: 'new-channel' });
    mockChannelsApi.create.mockResolvedValue({ channel: newChannel });

    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const { result } = renderHook(() => useCreateChannel('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: 'new-channel', type: 'public' });
    });

    expect(mockChannelsApi.create).toHaveBeenCalledWith('ws-1', {
      name: 'new-channel',
      type: 'public',
    });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['channels', 'ws-1'] });
  });
});

describe('useCreateDM', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates a DM channel', async () => {
    const queryClient = createTestQueryClient();
    const dmChannel = createMockChannel({ id: 'dm-1', name: '', type: 'dm' });
    mockChannelsApi.createDM.mockResolvedValue({ channel: dmChannel });

    const { result } = renderHook(() => useCreateDM('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ user_ids: ['user-1', 'user-2'] });
    });

    expect(mockChannelsApi.createDM).toHaveBeenCalledWith('ws-1', {
      user_ids: ['user-1', 'user-2'],
    });
  });
});

describe('useJoinChannel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('joins a channel and invalidates cache', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.join.mockResolvedValue({ success: true });

    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const { result } = renderHook(() => useJoinChannel('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('ch-1');
    });

    expect(mockChannelsApi.join).toHaveBeenCalledWith('ch-1');
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['channels', 'ws-1'] });
  });
});

describe('useLeaveChannel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('leaves a channel and invalidates cache', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.leave.mockResolvedValue({ success: true });

    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const { result } = renderHook(() => useLeaveChannel('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('ch-1');
    });

    expect(mockChannelsApi.leave).toHaveBeenCalledWith('ch-1');
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['channels', 'ws-1'] });
  });
});

describe('useMarkChannelAsRead', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls markAsRead API with correct parameters', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.markAsRead.mockResolvedValue({ last_read_message_id: 'msg-100' });

    const { result } = renderHook(() => useMarkChannelAsRead('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ channelId: 'ch-1', messageId: 'msg-50' });
    });

    expect(mockChannelsApi.markAsRead).toHaveBeenCalledWith('ch-1', 'msg-50');
  });

  it('returns last_read_message_id on success', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.markAsRead.mockResolvedValue({ last_read_message_id: 'msg-100' });

    const { result } = renderHook(() => useMarkChannelAsRead('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync({ channelId: 'ch-1' });
    });

    expect(response).toEqual({ last_read_message_id: 'msg-100' });
  });
});

describe('useMarkAllChannelsAsRead', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls markAllAsRead API', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.markAllAsRead.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useMarkAllChannelsAsRead('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync();
    });

    expect(mockChannelsApi.markAllAsRead).toHaveBeenCalledWith('ws-1');
  });

  it('returns success on completion', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.markAllAsRead.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useMarkAllChannelsAsRead('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync();
    });

    expect(response).toEqual({ success: true });
  });
});

describe('useStarChannel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls star API with channel ID', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.star.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useStarChannel('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('ch-1');
    });

    expect(mockChannelsApi.star).toHaveBeenCalledWith('ch-1');
  });

  it('returns success on completion', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.star.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useStarChannel('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync('ch-1');
    });

    expect(response).toEqual({ success: true });
  });
});

describe('useUnstarChannel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls unstar API with channel ID', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.unstar.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useUnstarChannel('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('ch-1');
    });

    expect(mockChannelsApi.unstar).toHaveBeenCalledWith('ch-1');
  });

  it('returns success on completion', async () => {
    const queryClient = createTestQueryClient();
    mockChannelsApi.unstar.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useUnstarChannel('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync('ch-1');
    });

    expect(response).toEqual({ success: true });
  });
});

describe('useUpdateChannel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls update API with correct parameters', async () => {
    const queryClient = createTestQueryClient();
    const updatedChannel = { id: 'ch-1', name: 'renamed-channel', description: 'New description' };
    mockChannelsApi.update.mockResolvedValue({ channel: updatedChannel });

    const { result } = renderHook(() => useUpdateChannel('ws-1', 'ch-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: 'renamed-channel', description: 'New description' });
    });

    expect(mockChannelsApi.update).toHaveBeenCalledWith('ch-1', {
      name: 'renamed-channel',
      description: 'New description',
    });
  });

  it('returns updated channel on success', async () => {
    const queryClient = createTestQueryClient();
    const updatedChannel = { id: 'ch-1', name: 'new-name' };
    mockChannelsApi.update.mockResolvedValue({ channel: updatedChannel });

    const { result } = renderHook(() => useUpdateChannel('ws-1', 'ch-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync({ name: 'new-name' });
    });

    expect(response).toEqual({ channel: updatedChannel });
  });
});

describe('useChannelMembers', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches channel members', async () => {
    const members = [
      { user_id: 'user-1', display_name: 'User 1', role: 'admin' },
      { user_id: 'user-2', display_name: 'User 2', role: 'member' },
    ];
    mockChannelsApi.listMembers.mockResolvedValue({ members });

    const { result } = renderHook(() => useChannelMembers('ch-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.members).toEqual(members);
    expect(mockChannelsApi.listMembers).toHaveBeenCalledWith('ch-1');
  });

  it('does not fetch when channelId is undefined', () => {
    const { result } = renderHook(() => useChannelMembers(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockChannelsApi.listMembers).not.toHaveBeenCalled();
  });
});
