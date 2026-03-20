import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

// Hoist mocks
const mockUseWorkspaceMembers = vi.hoisted(() => vi.fn());

vi.mock('@enzyme/shared/hooks/useWorkspaces', () => ({
  useWorkspaceMembers: mockUseWorkspaceMembers,
}));

import { useMentions } from './useMentions';

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

function createWrapper() {
  const client = createTestQueryClient();
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

const mockMembers = [
  { user_id: 'user-1', display_name: 'Alice', avatar_url: null },
  { user_id: 'user-2', display_name: 'Bob', avatar_url: 'https://example.com/bob.jpg' },
  { user_id: 'user-3', display_name: 'Charlie', avatar_url: null },
];

describe('useMentions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseWorkspaceMembers.mockReturnValue({ data: { members: mockMembers } });
  });

  it('returns null trigger when no @ in content', () => {
    const { result } = renderHook(() => useMentions('ws-1', 'Hello world', 11), {
      wrapper: createWrapper(),
    });

    expect(result.current.trigger).toBeNull();
    expect(result.current.options).toEqual([]);
  });

  it('returns trigger with query when @ detected', () => {
    const { result } = renderHook(() => useMentions('ws-1', 'Hello @al', 9), {
      wrapper: createWrapper(),
    });

    expect(result.current.trigger).not.toBeNull();
    expect(result.current.trigger?.isActive).toBe(true);
    expect(result.current.trigger?.query).toBe('al');
  });

  it('filters members by query', () => {
    const { result } = renderHook(() => useMentions('ws-1', 'Hello @ali', 10), {
      wrapper: createWrapper(),
    });

    // Should only match Alice
    const userOptions = result.current.options.filter((o) => o.type === 'user');
    expect(userOptions).toHaveLength(1);
    expect(userOptions[0].displayName).toBe('Alice');
  });

  it('includes special mentions (@here, @channel, @everyone)', () => {
    const { result } = renderHook(() => useMentions('ws-1', '@', 1), { wrapper: createWrapper() });

    // Should include special mentions
    const specialOptions = result.current.options.filter((o) => o.type === 'special');
    expect(specialOptions.length).toBeGreaterThanOrEqual(3);

    const specialIds = specialOptions.map((o) => o.id);
    expect(specialIds).toContain('here');
    expect(specialIds).toContain('channel');
    expect(specialIds).toContain('everyone');
  });

  it('filters special mentions by query', () => {
    const { result } = renderHook(() => useMentions('ws-1', '@he', 3), {
      wrapper: createWrapper(),
    });

    // Should match "here" from special mentions
    const specialOptions = result.current.options.filter((o) => o.type === 'special');
    expect(specialOptions.some((o) => o.id === 'here')).toBe(true);
  });

  it('returns empty options when trigger is not active', () => {
    const { result } = renderHook(() => useMentions('ws-1', 'Hello world', 11), {
      wrapper: createWrapper(),
    });

    expect(result.current.options).toEqual([]);
  });

  it('handles empty members list', () => {
    mockUseWorkspaceMembers.mockReturnValue({ data: { members: [] } });

    const { result } = renderHook(() => useMentions('ws-1', '@', 1), { wrapper: createWrapper() });

    // Should still include special mentions
    expect(result.current.options.length).toBeGreaterThanOrEqual(3);

    const userOptions = result.current.options.filter((o) => o.type === 'user');
    expect(userOptions).toHaveLength(0);
  });

  it('sorts exact matches first', () => {
    mockUseWorkspaceMembers.mockReturnValue({
      data: {
        members: [
          { user_id: 'user-1', display_name: 'Charlie', avatar_url: null },
          { user_id: 'user-2', display_name: 'Charles', avatar_url: null },
          { user_id: 'user-3', display_name: 'Char', avatar_url: null },
        ],
      },
    });

    const { result } = renderHook(() => useMentions('ws-1', '@char', 5), {
      wrapper: createWrapper(),
    });

    // "Char" should come first as it starts with "char" exactly
    const userOptions = result.current.options.filter((o) => o.type === 'user');
    expect(userOptions[0].displayName).toBe('Char');
  });

  it('handles missing members data', () => {
    mockUseWorkspaceMembers.mockReturnValue({ data: undefined });

    const { result } = renderHook(() => useMentions('ws-1', '@al', 3), {
      wrapper: createWrapper(),
    });

    // Should still work but with no user options
    const userOptions = result.current.options.filter((o) => o.type === 'user');
    expect(userOptions).toHaveLength(0);
  });
});
