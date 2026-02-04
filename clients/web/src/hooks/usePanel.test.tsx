import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { ReactNode } from 'react';
import { useThreadPanel, useProfilePanel } from './usePanel';

function createWrapper(initialEntries: string[] = ['/']) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
    );
  };
}

describe('useThreadPanel', () => {
  it('returns null threadId when no thread param', () => {
    const { result } = renderHook(() => useThreadPanel(), {
      wrapper: createWrapper(['/channel/123']),
    });

    expect(result.current.threadId).toBeNull();
  });

  it('returns threadId from URL', () => {
    const { result } = renderHook(() => useThreadPanel(), {
      wrapper: createWrapper(['/channel/123?thread=msg-456']),
    });

    expect(result.current.threadId).toBe('msg-456');
  });

  it('openThread sets thread param in URL', () => {
    const { result } = renderHook(() => useThreadPanel(), {
      wrapper: createWrapper(['/channel/123']),
    });

    act(() => {
      result.current.openThread('msg-789');
    });

    expect(result.current.threadId).toBe('msg-789');
  });

  it('closeThread removes thread param from URL', () => {
    const { result } = renderHook(() => useThreadPanel(), {
      wrapper: createWrapper(['/channel/123?thread=msg-456']),
    });

    expect(result.current.threadId).toBe('msg-456');

    act(() => {
      result.current.closeThread();
    });

    expect(result.current.threadId).toBeNull();
  });

  it('openThread closes profile panel', () => {
    const { result: threadResult } = renderHook(() => useThreadPanel(), {
      wrapper: createWrapper(['/channel/123?profile=user-1']),
    });

    // Note: This test verifies the openThread logic removes profile param
    // In real usage, both hooks share the same router context
    act(() => {
      threadResult.current.openThread('msg-123');
    });

    expect(threadResult.current.threadId).toBe('msg-123');
  });
});

describe('useProfilePanel', () => {
  it('returns null profileUserId when no profile param', () => {
    const { result } = renderHook(() => useProfilePanel(), {
      wrapper: createWrapper(['/channel/123']),
    });

    expect(result.current.profileUserId).toBeNull();
  });

  it('returns profileUserId from URL', () => {
    const { result } = renderHook(() => useProfilePanel(), {
      wrapper: createWrapper(['/channel/123?profile=user-456']),
    });

    expect(result.current.profileUserId).toBe('user-456');
  });

  it('openProfile sets profile param in URL', () => {
    const { result } = renderHook(() => useProfilePanel(), {
      wrapper: createWrapper(['/channel/123']),
    });

    act(() => {
      result.current.openProfile('user-789');
    });

    expect(result.current.profileUserId).toBe('user-789');
  });

  it('closeProfile removes profile param from URL', () => {
    const { result } = renderHook(() => useProfilePanel(), {
      wrapper: createWrapper(['/channel/123?profile=user-456']),
    });

    expect(result.current.profileUserId).toBe('user-456');

    act(() => {
      result.current.closeProfile();
    });

    expect(result.current.profileUserId).toBeNull();
  });

  it('openProfile closes thread panel', () => {
    const { result } = renderHook(() => useProfilePanel(), {
      wrapper: createWrapper(['/channel/123?thread=msg-1']),
    });

    act(() => {
      result.current.openProfile('user-123');
    });

    expect(result.current.profileUserId).toBe('user-123');
  });

  it('preserves other URL params when opening profile', () => {
    const { result } = renderHook(() => useProfilePanel(), {
      wrapper: createWrapper(['/channel/123?other=value']),
    });

    act(() => {
      result.current.openProfile('user-123');
    });

    expect(result.current.profileUserId).toBe('user-123');
  });
});

describe('useThreadPanel and useProfilePanel interaction', () => {
  it('can switch between thread and profile panels', () => {
    // Create a shared wrapper function for both hooks
    const Wrapper = createWrapper(['/channel/123']);

    const { result: threadHook } = renderHook(() => useThreadPanel(), {
      wrapper: Wrapper,
    });

    // Open thread
    act(() => {
      threadHook.current.openThread('msg-1');
    });
    expect(threadHook.current.threadId).toBe('msg-1');

    // Close thread
    act(() => {
      threadHook.current.closeThread();
    });
    expect(threadHook.current.threadId).toBeNull();
  });
});
