import { describe, it, expect } from 'vitest';
import { isEmojiOnly } from './isEmojiOnly';
import type { MrkdwnSegment } from './parser';

describe('isEmojiOnly', () => {
  it('returns true for a single emoji shortcode', () => {
    const segments: MrkdwnSegment[] = [{ type: 'emoji_shortcode', name: 'thumbsup' }];
    expect(isEmojiOnly(segments)).toBe(true);
  });

  it('returns true for two emoji shortcodes', () => {
    const segments: MrkdwnSegment[] = [
      { type: 'emoji_shortcode', name: 'fire' },
      { type: 'emoji_shortcode', name: 'heart' },
    ];
    expect(isEmojiOnly(segments)).toBe(true);
  });

  it('returns true for three emoji shortcodes', () => {
    const segments: MrkdwnSegment[] = [
      { type: 'emoji_shortcode', name: 'fire' },
      { type: 'emoji_shortcode', name: 'fire' },
      { type: 'emoji_shortcode', name: 'fire' },
    ];
    expect(isEmojiOnly(segments)).toBe(true);
  });

  it('returns false for four emoji shortcodes', () => {
    const segments: MrkdwnSegment[] = [
      { type: 'emoji_shortcode', name: 'wave' },
      { type: 'emoji_shortcode', name: 'wave' },
      { type: 'emoji_shortcode', name: 'wave' },
      { type: 'emoji_shortcode', name: 'wave' },
    ];
    expect(isEmojiOnly(segments)).toBe(false);
  });

  it('returns true for emoji with whitespace between', () => {
    const segments: MrkdwnSegment[] = [
      { type: 'emoji_shortcode', name: 'wave' },
      { type: 'text', content: ' ' },
      { type: 'emoji_shortcode', name: 'smile' },
    ];
    expect(isEmojiOnly(segments)).toBe(true);
  });

  it('returns false for mixed emoji and text', () => {
    const segments: MrkdwnSegment[] = [
      { type: 'text', content: 'hello ' },
      { type: 'emoji_shortcode', name: 'wave' },
    ];
    expect(isEmojiOnly(segments)).toBe(false);
  });

  it('returns false for only text', () => {
    const segments: MrkdwnSegment[] = [{ type: 'text', content: 'hello' }];
    expect(isEmojiOnly(segments)).toBe(false);
  });

  it('returns false for empty segments', () => {
    expect(isEmojiOnly([])).toBe(false);
  });

  it('returns true for emoji with line breaks', () => {
    const segments: MrkdwnSegment[] = [
      { type: 'emoji_shortcode', name: 'wave' },
      { type: 'line_break' },
      { type: 'emoji_shortcode', name: 'smile' },
    ];
    expect(isEmojiOnly(segments)).toBe(true);
  });

  it('returns false for emoji with line breaks exceeding 3 emoji', () => {
    const segments: MrkdwnSegment[] = [
      { type: 'emoji_shortcode', name: 'a' },
      { type: 'line_break' },
      { type: 'emoji_shortcode', name: 'b' },
      { type: 'emoji_shortcode', name: 'c' },
      { type: 'emoji_shortcode', name: 'd' },
    ];
    expect(isEmojiOnly(segments)).toBe(false);
  });
});
