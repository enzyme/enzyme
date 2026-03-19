import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  formatTime,
  formatDate,
  formatRelativeTime,
  getInitials,
  getAvatarColor,
  groupReactions,
  debounce,
} from './utils';

describe('formatTime', () => {
  it('returns formatted time', () => {
    // Use a fixed date to ensure consistent test results
    const date = '2024-01-15T14:30:00Z';
    const result = formatTime(date);
    // The output depends on locale, but should contain hours and minutes
    expect(result).toMatch(/\d{1,2}:\d{2}/);
  });

  it('handles different times of day', () => {
    const morning = formatTime('2024-01-15T08:00:00Z');
    const evening = formatTime('2024-01-15T20:00:00Z');
    // Both should be valid time strings
    expect(morning).toMatch(/\d{1,2}:\d{2}/);
    expect(evening).toMatch(/\d{1,2}:\d{2}/);
  });
});

describe('formatDate', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns "Today" for today\'s date', () => {
    vi.setSystemTime(new Date('2024-01-15T12:00:00Z'));
    const result = formatDate('2024-01-15T08:00:00Z');
    expect(result).toBe('Today');
  });

  it('returns "Yesterday" for yesterday\'s date', () => {
    vi.setSystemTime(new Date('2024-01-15T12:00:00Z'));
    const result = formatDate('2024-01-14T08:00:00Z');
    expect(result).toBe('Yesterday');
  });

  it('returns formatted date for older dates', () => {
    vi.setSystemTime(new Date('2024-01-15T12:00:00Z'));
    const result = formatDate('2024-01-10T08:00:00Z');
    // Should contain month, day, and year
    expect(result).toMatch(/Jan\s+10,\s+2024/);
  });
});

describe('formatRelativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns "just now" for less than 1 minute', () => {
    vi.setSystemTime(new Date('2024-01-15T12:00:30Z'));
    const result = formatRelativeTime('2024-01-15T12:00:00Z');
    expect(result).toBe('just now');
  });

  it('returns minutes ago for less than 1 hour', () => {
    vi.setSystemTime(new Date('2024-01-15T12:05:00Z'));
    const result = formatRelativeTime('2024-01-15T12:00:00Z');
    expect(result).toBe('5m ago');
  });

  it('returns hours ago for less than 24 hours', () => {
    vi.setSystemTime(new Date('2024-01-15T14:00:00Z'));
    const result = formatRelativeTime('2024-01-15T12:00:00Z');
    expect(result).toBe('2h ago');
  });

  it('returns days ago for less than 7 days', () => {
    vi.setSystemTime(new Date('2024-01-18T12:00:00Z'));
    const result = formatRelativeTime('2024-01-15T12:00:00Z');
    expect(result).toBe('3d ago');
  });

  it('falls back to formatDate for more than 7 days', () => {
    vi.setSystemTime(new Date('2024-01-25T12:00:00Z'));
    const result = formatRelativeTime('2024-01-15T12:00:00Z');
    // Should return formatted date
    expect(result).toMatch(/Jan\s+15,\s+2024/);
  });
});

describe('getInitials', () => {
  it('returns single initial for single word', () => {
    expect(getInitials('John')).toBe('J');
  });

  it('returns two initials for two words', () => {
    expect(getInitials('John Doe')).toBe('JD');
  });

  it('truncates to 2 initials for three or more words', () => {
    expect(getInitials('John Middle Doe')).toBe('JM');
  });

  it('handles empty string', () => {
    expect(getInitials('')).toBe('');
  });

  it('converts to uppercase', () => {
    expect(getInitials('john doe')).toBe('JD');
  });
});

describe('getAvatarColor', () => {
  it('returns consistent color for same ID', () => {
    const color1 = getAvatarColor('user-123');
    const color2 = getAvatarColor('user-123');
    expect(color1).toBe(color2);
  });

  it('returns valid Tailwind color class', () => {
    const color = getAvatarColor('user-123');
    expect(color).toMatch(/^bg-[a-z]+-500$/);
  });

  it('different IDs can return different colors', () => {
    // Test with enough IDs to ensure at least one is different
    const ids = ['user-1', 'user-2', 'user-3', 'user-abc', 'user-xyz'];
    const colors = ids.map(getAvatarColor);
    const uniqueColors = new Set(colors);
    // Should have at least 2 different colors among 5 IDs
    expect(uniqueColors.size).toBeGreaterThan(1);
  });
});

describe('groupReactions', () => {
  it('groups reactions by emoji', () => {
    const reactions = [
      { emoji: '👍', user_id: 'user1' },
      { emoji: '👍', user_id: 'user2' },
      { emoji: '❤️', user_id: 'user1' },
    ];
    const grouped = groupReactions(reactions);

    expect(grouped.get('👍')).toEqual(['user1', 'user2']);
    expect(grouped.get('❤️')).toEqual(['user1']);
  });

  it('handles multiple users for same emoji', () => {
    const reactions = [
      { emoji: '🔥', user_id: 'user1' },
      { emoji: '🔥', user_id: 'user2' },
      { emoji: '🔥', user_id: 'user3' },
    ];
    const grouped = groupReactions(reactions);

    expect(grouped.get('🔥')).toEqual(['user1', 'user2', 'user3']);
  });

  it('handles empty array', () => {
    const grouped = groupReactions([]);
    expect(grouped.size).toBe(0);
  });
});

describe('debounce', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('delays function execution', () => {
    const fn = vi.fn();
    const debouncedFn = debounce(fn, 100);

    debouncedFn();
    expect(fn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('cancels previous pending calls', () => {
    const fn = vi.fn();
    const debouncedFn = debounce(fn, 100);

    debouncedFn();
    vi.advanceTimersByTime(50);
    debouncedFn(); // Reset the timer
    vi.advanceTimersByTime(50);
    expect(fn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(50);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('passes arguments to the function', () => {
    const fn = vi.fn();
    const debouncedFn = debounce(fn, 100);

    debouncedFn('arg1', 'arg2');
    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledWith('arg1', 'arg2');
  });
});
