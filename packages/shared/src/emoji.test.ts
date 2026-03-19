import { describe, it, expect } from 'vitest';
import { searchEmojis, COMMON_EMOJIS } from './emoji';

describe('searchEmojis', () => {
  it('returns common emojis for empty query', () => {
    const results = searchEmojis('');
    expect(results.length).toBeGreaterThan(0);
    // Should return common emojis
    expect(results.map((r) => r.emoji)).toEqual(expect.arrayContaining([COMMON_EMOJIS[0]]));
    // Every result should have a resolved shortcode
    for (const r of results) {
      expect(r.shortcode).toBeTruthy();
    }
  });

  it('returns correct shortcodes for common emojis', () => {
    const results = searchEmojis('');
    const thumbsUp = results.find((r) => r.emoji === '👍');
    expect(thumbsUp).toBeDefined();
    expect(thumbsUp!.shortcode).toBe('+1');
  });

  it('exact match scores highest', () => {
    const results = searchEmojis('fire');
    expect(results[0].shortcode).toBe('fire');
    expect(results[0].emoji).toBe('🔥');
  });

  it('prefix match scores higher than substring match', () => {
    const results = searchEmojis('th', 24);
    // Prefix matches like 'thumbsup' should appear before substring matches like 'athletic_shoe'
    const thumbsIndex = results.findIndex((r) => r.shortcode === 'thumbsup');
    const substringIndex = results.findIndex((r) => r.shortcode === 'athletic_shoe');
    expect(thumbsIndex).not.toBe(-1);
    expect(substringIndex).not.toBe(-1);
    expect(thumbsIndex).toBeLessThan(substringIndex);
  });

  it('substring match scores lowest', () => {
    const results = searchEmojis('ear');
    // 'ear' exact match should be first if it exists, or prefix matches
    // Substring matches (e.g., 'bear', 'pear') should come after
    const earIndex = results.findIndex((r) => r.shortcode === 'ear');
    const bearIndex = results.findIndex((r) => r.shortcode === 'bear');
    if (earIndex !== -1 && bearIndex !== -1) {
      expect(earIndex).toBeLessThan(bearIndex);
    }
  });

  it('respects limit parameter', () => {
    const results = searchEmojis('a', 3);
    expect(results.length).toBeLessThanOrEqual(3);
  });

  it('returns empty array for no matches', () => {
    const results = searchEmojis('xyznonexistent');
    expect(results).toEqual([]);
  });

  it('is case insensitive', () => {
    const lowerResults = searchEmojis('fire');
    const upperResults = searchEmojis('FIRE');
    expect(lowerResults).toEqual(upperResults);
  });
});
