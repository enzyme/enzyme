import { describe, it, expect } from 'vitest';
import { fuzzyMatch } from './fuzzyMatch';

describe('fuzzyMatch', () => {
  describe('exact substring matches', () => {
    it('matches exact string', () => {
      const result = fuzzyMatch('general', 'general');
      expect(result.matches).toBe(true);
    });

    it('matches substring', () => {
      const result = fuzzyMatch('gen', 'general');
      expect(result.matches).toBe(true);
    });

    it('is case insensitive', () => {
      const result = fuzzyMatch('Gen', 'general');
      expect(result.matches).toBe(true);
    });

    it('scores prefix matches higher than mid-string matches', () => {
      const prefix = fuzzyMatch('gen', 'general');
      const midString = fuzzyMatch('era', 'general');
      expect(prefix.score).toBeGreaterThan(midString.score);
    });

    it('scores word-boundary matches higher than mid-word matches', () => {
      const wordBoundary = fuzzyMatch('bob', 'hey bob');
      const midWord = fuzzyMatch('bob', 'snobby');
      expect(wordBoundary.score).toBeGreaterThan(midWord.score);
    });
  });

  describe('sequential character matching', () => {
    it('matches characters in order', () => {
      const result = fuzzyMatch('gnrl', 'general');
      expect(result.matches).toBe(true);
    });

    it('does not match when characters are out of order', () => {
      const result = fuzzyMatch('ba', 'abc');
      expect(result.matches).toBe(false);
    });

    it('does not match when query has characters not in text', () => {
      const result = fuzzyMatch('xyz', 'general');
      expect(result.matches).toBe(false);
    });

    it('scores consecutive character matches higher', () => {
      const consecutive = fuzzyMatch('gen', 'a-general');
      const scattered = fuzzyMatch('gnl', 'general');
      expect(consecutive.score).toBeGreaterThan(scattered.score);
    });
  });

  describe('scoring priorities', () => {
    it('scores exact substring higher than sequential match', () => {
      const substring = fuzzyMatch('ace', 'Grace Patel');
      const sequential = fuzzyMatch('gpl', 'Grace Patel');
      expect(substring.score).toBeGreaterThan(sequential.score);
    });

    it('returns score 0 for non-matches', () => {
      const result = fuzzyMatch('zzz', 'general');
      expect(result.matches).toBe(false);
      expect(result.score).toBe(0);
    });
  });

  describe('edge cases', () => {
    it('matches single character', () => {
      const result = fuzzyMatch('g', 'general');
      expect(result.matches).toBe(true);
    });

    it('handles query equal to text', () => {
      const result = fuzzyMatch('general', 'general');
      expect(result.matches).toBe(true);
    });

    it('does not match when query is longer than text', () => {
      const result = fuzzyMatch('generals', 'general');
      expect(result.matches).toBe(false);
    });

    it('handles spaces in query and text', () => {
      const result = fuzzyMatch('grace p', 'Grace Patel');
      expect(result.matches).toBe(true);
    });

    it('handles empty text', () => {
      const result = fuzzyMatch('a', '');
      expect(result.matches).toBe(false);
    });
  });
});
