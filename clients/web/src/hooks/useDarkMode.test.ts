import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDarkMode } from './useDarkMode';

describe('useDarkMode', () => {
  const mockMatchMedia = (matches: boolean) => {
    const listeners: Array<(e: MediaQueryListEvent) => void> = [];
    return vi.fn().mockImplementation(() => ({
      matches,
      media: '(prefers-color-scheme: dark)',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: (_: string, handler: (e: MediaQueryListEvent) => void) => {
        listeners.push(handler);
      },
      removeEventListener: (_: string, handler: (e: MediaQueryListEvent) => void) => {
        const idx = listeners.indexOf(handler);
        if (idx > -1) listeners.splice(idx, 1);
      },
      dispatchEvent: vi.fn(),
      // Helper to trigger change event in tests
      _triggerChange: (newMatches: boolean) => {
        listeners.forEach((fn) =>
          fn({ matches: newMatches } as MediaQueryListEvent)
        );
      },
    }));
  };

  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove('dark');
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove('dark');
  });

  it('follows system preference by default', () => {
    window.matchMedia = mockMatchMedia(true);

    const { result } = renderHook(() => useDarkMode());

    expect(result.current.mode).toBe('system');
    expect(result.current.darkMode).toBe(true);
  });

  it('follows system preference when set to light', () => {
    window.matchMedia = mockMatchMedia(false);

    const { result } = renderHook(() => useDarkMode());

    expect(result.current.mode).toBe('system');
    expect(result.current.darkMode).toBe(false);
  });

  it('toggle switches from system dark to light', () => {
    window.matchMedia = mockMatchMedia(true);

    const { result } = renderHook(() => useDarkMode());

    expect(result.current.darkMode).toBe(true);

    act(() => {
      result.current.toggle();
    });

    expect(result.current.darkMode).toBe(false);
    expect(result.current.mode).toBe('light');
  });

  it('toggle switches from system light to dark', () => {
    window.matchMedia = mockMatchMedia(false);

    const { result } = renderHook(() => useDarkMode());

    expect(result.current.darkMode).toBe(false);

    act(() => {
      result.current.toggle();
    });

    expect(result.current.darkMode).toBe(true);
    expect(result.current.mode).toBe('dark');
  });

  it('toggle switches between explicit dark and light', () => {
    window.matchMedia = mockMatchMedia(false);

    const { result } = renderHook(() => useDarkMode());

    // Set to dark explicitly
    act(() => {
      result.current.setDarkMode(true);
    });

    expect(result.current.darkMode).toBe(true);
    expect(result.current.mode).toBe('dark');

    // Toggle should switch to light
    act(() => {
      result.current.toggle();
    });

    expect(result.current.darkMode).toBe(false);
    expect(result.current.mode).toBe('light');
  });

  it('setDarkMode sets explicit dark mode', () => {
    window.matchMedia = mockMatchMedia(false);

    const { result } = renderHook(() => useDarkMode());

    act(() => {
      result.current.setDarkMode(true);
    });

    expect(result.current.darkMode).toBe(true);
    expect(result.current.mode).toBe('dark');
  });

  it('setDarkMode sets explicit light mode', () => {
    window.matchMedia = mockMatchMedia(true);

    const { result } = renderHook(() => useDarkMode());

    act(() => {
      result.current.setDarkMode(false);
    });

    expect(result.current.darkMode).toBe(false);
    expect(result.current.mode).toBe('light');
  });

  it('setMode switches to system preference', () => {
    window.matchMedia = mockMatchMedia(true);

    const { result } = renderHook(() => useDarkMode());

    // First set to explicit light
    act(() => {
      result.current.setMode('light');
    });

    expect(result.current.mode).toBe('light');
    expect(result.current.darkMode).toBe(false);

    // Then switch to system
    act(() => {
      result.current.setMode('system');
    });

    expect(result.current.mode).toBe('system');
    expect(result.current.darkMode).toBe(true); // System prefers dark
  });

  it('setMode switches to dark', () => {
    window.matchMedia = mockMatchMedia(false);

    const { result } = renderHook(() => useDarkMode());

    act(() => {
      result.current.setMode('dark');
    });

    expect(result.current.mode).toBe('dark');
    expect(result.current.darkMode).toBe(true);
  });

  it('setMode switches to light', () => {
    window.matchMedia = mockMatchMedia(true);

    const { result } = renderHook(() => useDarkMode());

    act(() => {
      result.current.setMode('light');
    });

    expect(result.current.mode).toBe('light');
    expect(result.current.darkMode).toBe(false);
  });

  it('adds dark class to document when dark mode is active', () => {
    window.matchMedia = mockMatchMedia(true);

    renderHook(() => useDarkMode());

    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('removes dark class from document when light mode is active', () => {
    window.matchMedia = mockMatchMedia(false);
    document.documentElement.classList.add('dark');

    renderHook(() => useDarkMode());

    expect(document.documentElement.classList.contains('dark')).toBe(false);
  });

  it('persists preference to localStorage', () => {
    window.matchMedia = mockMatchMedia(false);

    const { result } = renderHook(() => useDarkMode());

    act(() => {
      result.current.setDarkMode(true);
    });

    // Check localStorage was updated
    const stored = localStorage.getItem('feather:dark-mode');
    expect(stored).toBe('true');
  });

  it('restores preference from localStorage', () => {
    localStorage.setItem('feather:dark-mode', 'true');
    window.matchMedia = mockMatchMedia(false);

    const { result } = renderHook(() => useDarkMode());

    expect(result.current.darkMode).toBe(true);
    expect(result.current.mode).toBe('dark');
  });
});
