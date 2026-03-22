import { describe, it, expect } from 'vitest';
import { buildSuggestions } from './buildSuggestions';

const members = [
  { user_id: 'u1', display_name: 'Alice Chen', avatar_url: null },
  { user_id: 'u2', display_name: 'Bob Martinez', avatar_url: null },
  { user_id: 'u3', display_name: 'Carol Williams', avatar_url: null },
];

const channels = [
  { id: 'ch1', name: 'general', type: 'public' },
  { id: 'ch2', name: 'random', type: 'public' },
  { id: 'ch3', name: 'dm-alice-bob', type: 'dm' },
  { id: 'ch4', name: 'engineering', type: 'private' },
];

describe('buildSuggestions', () => {
  describe('no trigger', () => {
    it('returns empty when no trigger character present', () => {
      expect(buildSuggestions('hello', 5, members, channels)).toEqual([]);
    });

    it('returns empty for empty text', () => {
      expect(buildSuggestions('', 0, members, channels)).toEqual([]);
    });

    it('returns empty when trigger is mid-word', () => {
      expect(buildSuggestions('email@example', 13, members, channels)).toEqual([]);
    });
  });

  describe('@ mentions', () => {
    it('shows special mentions and all members with no query', () => {
      const results = buildSuggestions('@', 1, members, channels);
      expect(results.length).toBeGreaterThan(0);
      // Special mentions come first
      expect(results[0].id).toBe('special-here');
    });

    it('filters members by query and includes fuzzy matches', () => {
      const results = buildSuggestions('@alice', 6, members, channels);
      const memberResults = results.filter((r) => !r.id.startsWith('special-'));
      expect(memberResults.length).toBeGreaterThanOrEqual(1);
      expect(memberResults[0].displayText).toBe('Alice Chen');
      expect(memberResults[0].token).toBe('<@u1>');
    });

    it('generates correct token format', () => {
      const results = buildSuggestions('@bob', 4, members, channels);
      const bob = results.find((r) => r.id === 'u2');
      expect(bob).toBeDefined();
      expect(bob!.token).toBe('<@u2>');
    });

    it('includes special mentions matching query', () => {
      const results = buildSuggestions('@here', 5, members, channels);
      expect(results.some((r) => r.id === 'special-here')).toBe(true);
    });

    it('limits results to 5', () => {
      const manyMembers = Array.from({ length: 10 }, (_, i) => ({
        user_id: `u${i}`,
        display_name: `User ${i}`,
        avatar_url: null,
      }));
      const results = buildSuggestions('@', 1, manyMembers, channels);
      expect(results.length).toBeLessThanOrEqual(5);
    });

    it('works with @ after a space', () => {
      const results = buildSuggestions('hello @alice', 12, members, channels);
      const memberResults = results.filter((r) => !r.id.startsWith('special-'));
      expect(memberResults.length).toBeGreaterThanOrEqual(1);
      expect(memberResults[0].displayText).toBe('Alice Chen');
    });
  });

  describe('# channel mentions', () => {
    it('shows all non-DM channels with no query', () => {
      const results = buildSuggestions('#', 1, members, channels);
      expect(results.every((r) => r.id !== 'ch3')).toBe(true);
      expect(results).toHaveLength(3);
    });

    it('filters channels by query', () => {
      const results = buildSuggestions('#gen', 4, members, channels);
      expect(results.length).toBeGreaterThanOrEqual(1);
      expect(results[0].displayText).toBe('general');
      expect(results[0].token).toBe('<#ch1>');
    });

    it('excludes DM and group_dm channels', () => {
      const withGroupDm = [...channels, { id: 'ch5', name: 'group-chat', type: 'group_dm' }];
      const results = buildSuggestions('#', 1, members, withGroupDm);
      expect(results.every((r) => r.id !== 'ch3' && r.id !== 'ch5')).toBe(true);
    });

    it('returns empty when no channels provided', () => {
      const results = buildSuggestions('#gen', 4, members, undefined);
      expect(results).toEqual([]);
    });

    it('works with # after a space', () => {
      const results = buildSuggestions('see #gen', 8, members, channels);
      expect(results.length).toBeGreaterThanOrEqual(1);
      expect(results[0].displayText).toBe('general');
    });
  });

  describe(': emoji search', () => {
    it('returns empty when query is too short', () => {
      const results = buildSuggestions(':s', 2, members, channels);
      expect(results).toEqual([]);
    });

    it('returns emoji results for longer queries', () => {
      const results = buildSuggestions(':smile', 6, members, channels);
      expect(results.length).toBeGreaterThan(0);
      expect(results[0].token).toMatch(/^:.+:$/);
    });

    it('limits to 5 results', () => {
      const results = buildSuggestions(':face', 5, members, channels);
      expect(results.length).toBeLessThanOrEqual(5);
    });
  });
});
