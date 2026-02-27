import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useResizableWidth } from './useResizableWidth';

const KEY = 'enzyme:test-width';

describe('useResizableWidth', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('returns defaultWidth when localStorage is empty', () => {
    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.width).toBe(256);
  });

  it('reads initial width from localStorage', () => {
    localStorage.setItem(KEY, '300');

    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.width).toBe(300);
  });

  it('clamps stored value to min', () => {
    localStorage.setItem(KEY, '50');

    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.width).toBe(180);
  });

  it('clamps stored value to max', () => {
    localStorage.setItem(KEY, '999');

    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.width).toBe(400);
  });

  it('falls back to default for non-numeric localStorage value', () => {
    localStorage.setItem(KEY, 'not-a-number');

    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.width).toBe(256);
  });

  it('falls back to default for NaN localStorage value', () => {
    localStorage.setItem(KEY, 'NaN');

    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.width).toBe(256);
  });

  it('falls back to default for Infinity localStorage value', () => {
    localStorage.setItem(KEY, 'Infinity');

    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.width).toBe(256);
  });

  it('returns dividerProps with onPointerDown handler', () => {
    const { result } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    expect(result.current.dividerProps).toHaveProperty('onPointerDown');
    expect(typeof result.current.dividerProps.onPointerDown).toBe('function');
  });

  it('accepts left divider side parameter', () => {
    const { result } = renderHook(() => useResizableWidth(KEY, 384, 300, 600, 'left'));

    expect(result.current.width).toBe(384);
  });

  it('maintains stable onPointerDown reference across renders', () => {
    const { result, rerender } = renderHook(() => useResizableWidth(KEY, 256, 180, 400));

    const first = result.current.dividerProps.onPointerDown;
    rerender();
    const second = result.current.dividerProps.onPointerDown;

    expect(first).toBe(second);
  });
});
