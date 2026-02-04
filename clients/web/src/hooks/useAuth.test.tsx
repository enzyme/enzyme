import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { createMockUser, createMockWorkspaceSummary } from '../test-utils';

// Hoist the mock to avoid initialization issues
const mockAuthApi = vi.hoisted(() => ({
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  me: vi.fn(),
  forgotPassword: vi.fn(),
  resetPassword: vi.fn(),
}));

vi.mock('../api/auth', () => ({
  authApi: mockAuthApi,
}));

// Mock ApiError
vi.mock('@feather/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@feather/api-client')>();
  return {
    ...original,
    ApiError: class ApiError extends Error {
      status: number;
      constructor(message: string, status: number) {
        super(message);
        this.status = status;
      }
    },
  };
});

// Import after mocks
import { useAuth } from './useAuth';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns loading state initially', () => {
    mockAuthApi.me.mockImplementation(() => new Promise(() => {})); // Never resolves

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.user).toBeUndefined();
  });

  it('returns authenticated user after successful me() call', async () => {
    const user = createMockUser({ id: 'user-1', email: 'test@example.com' });
    const workspaces = [createMockWorkspaceSummary({ id: 'ws-1' })];
    mockAuthApi.me.mockResolvedValue({ user, workspaces });

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.user).toEqual(user);
    expect(result.current.workspaces).toEqual(workspaces);
    expect(result.current.error).toBeNull();
  });

  it('returns unauthenticated state on 401 error', async () => {
    const { ApiError } = await import('@feather/api-client');
    mockAuthApi.me.mockRejectedValue(new ApiError('UNAUTHORIZED', 'Unauthorized', 401));

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.user).toBeUndefined();
    expect(result.current.error).toBeNull(); // 401 is not treated as an error
  });

  it('exposes login mutation', async () => {
    const user = createMockUser();
    mockAuthApi.me.mockRejectedValue(new Error('Not logged in'));
    mockAuthApi.login.mockResolvedValue({ user });

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(typeof result.current.login).toBe('function');
    expect(result.current.isLoggingIn).toBe(false);
  });

  it('exposes logout mutation', async () => {
    const user = createMockUser();
    mockAuthApi.me.mockResolvedValue({ user, workspaces: [] });
    mockAuthApi.logout.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    expect(typeof result.current.logout).toBe('function');
    expect(result.current.isLoggingOut).toBe(false);
  });
});
