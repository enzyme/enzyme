import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import type { NotificationPreferences } from '@feather/api-client';

// Hoist mocks
const mockChannelsApi = vi.hoisted(() => ({
  getNotifications: vi.fn(),
  updateNotifications: vi.fn(),
}));

vi.mock('../api/channels', () => ({
  channelsApi: mockChannelsApi,
}));

import {
  useChannelNotifications,
  useUpdateChannelNotifications,
} from './useChannelNotifications';

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

describe('useChannelNotifications', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches notification settings', async () => {
    const preferences = { muted: false, level: 'all' };
    mockChannelsApi.getNotifications.mockResolvedValue({ preferences });

    const { result } = renderHook(() => useChannelNotifications('ch-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toEqual({ preferences });
    expect(mockChannelsApi.getNotifications).toHaveBeenCalledWith('ch-1');
  });

  it('is disabled when channelId is undefined', () => {
    const { result } = renderHook(() => useChannelNotifications(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockChannelsApi.getNotifications).not.toHaveBeenCalled();
  });

  it('returns loading state initially', () => {
    mockChannelsApi.getNotifications.mockImplementation(() => new Promise(() => {}));

    const { result } = renderHook(() => useChannelNotifications('ch-1'), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
  });
});

describe('useUpdateChannelNotifications', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls API with settings', async () => {
    const queryClient = createTestQueryClient();
    const newPreferences: NotificationPreferences = { notify_level: 'mentions', email_enabled: false };
    mockChannelsApi.updateNotifications.mockResolvedValue({ preferences: newPreferences });

    const { result } = renderHook(() => useUpdateChannelNotifications('ch-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(newPreferences);
    });

    expect(mockChannelsApi.updateNotifications).toHaveBeenCalledWith('ch-1', newPreferences);
  });

  it('invalidates cache after mutation', async () => {
    const queryClient = createTestQueryClient();
    const initialPreferences: NotificationPreferences = { notify_level: 'all', email_enabled: true };
    const newPreferences: NotificationPreferences = { notify_level: 'mentions', email_enabled: false };
    mockChannelsApi.updateNotifications.mockResolvedValue({ preferences: newPreferences });

    // Pre-populate cache
    queryClient.setQueryData(['channel-notifications', 'ch-1'], { preferences: initialPreferences });

    const { result } = renderHook(() => useUpdateChannelNotifications('ch-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(newPreferences);
    });

    // After mutation settles, API was called correctly
    expect(mockChannelsApi.updateNotifications).toHaveBeenCalledWith('ch-1', newPreferences);
  });

  it('returns updated preferences on success', async () => {
    const queryClient = createTestQueryClient();
    const newPreferences: NotificationPreferences = { notify_level: 'all', email_enabled: true };
    mockChannelsApi.updateNotifications.mockResolvedValue({ preferences: newPreferences });

    const { result } = renderHook(() => useUpdateChannelNotifications('ch-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync(newPreferences);
    });

    expect(response).toEqual({ preferences: newPreferences });
  });

  it('uses correct channel ID from hook parameter', async () => {
    const queryClient = createTestQueryClient();
    const preferences: NotificationPreferences = { notify_level: 'none', email_enabled: false };
    mockChannelsApi.updateNotifications.mockResolvedValue({ preferences });

    const { result } = renderHook(() => useUpdateChannelNotifications('specific-channel'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(preferences);
    });

    expect(mockChannelsApi.updateNotifications).toHaveBeenCalledWith('specific-channel', preferences);
  });
});
