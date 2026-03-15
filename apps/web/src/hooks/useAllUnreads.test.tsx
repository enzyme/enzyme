import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

const mockApiClient = vi.hoisted(() => ({
  POST: vi.fn(),
}));

vi.mock('@enzyme/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@enzyme/api-client')>();
  return { ...original, apiClient: mockApiClient };
});

import { useAllUnreads } from './useAllUnreads';
import { mockResponse } from '../test-utils/mocks/api-client';

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

describe('useAllUnreads', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches unreads with workspaceId', async () => {
    const unreadsResult = {
      messages: [{ id: 'msg-1', channel_id: 'ch-1', content: 'Unread message' }],
      next_cursor: undefined,
    };
    mockApiClient.POST.mockResolvedValue(mockResponse(unreadsResult));

    const { result } = renderHook(() => useAllUnreads({ workspaceId: 'ws-1' }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(mockApiClient.POST).toHaveBeenCalledWith('/workspaces/{wid}/unreads', {
      params: { path: { wid: 'ws-1' } },
      body: { limit: 50, cursor: undefined },
    });
    expect(result.current.data?.pages[0]).toEqual(unreadsResult);
  });

  it('supports pagination (infinite query)', async () => {
    const firstPage = {
      messages: [{ id: 'msg-1', content: 'First' }],
      next_cursor: 'cursor-1',
    };
    mockApiClient.POST.mockResolvedValue(mockResponse(firstPage));

    const { result } = renderHook(() => useAllUnreads({ workspaceId: 'ws-1' }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.hasNextPage).toBe(true);
  });

  it('disabled when workspaceId is empty', () => {
    const { result } = renderHook(() => useAllUnreads({ workspaceId: '' }), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockApiClient.POST).not.toHaveBeenCalled();
  });

  it('disabled when enabled option is false', () => {
    const { result } = renderHook(() => useAllUnreads({ workspaceId: 'ws-1', enabled: false }), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockApiClient.POST).not.toHaveBeenCalled();
  });

  it('returns loading state initially', () => {
    mockApiClient.POST.mockImplementation(() => new Promise(() => {}));

    const { result } = renderHook(() => useAllUnreads({ workspaceId: 'ws-1' }), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
  });
});
