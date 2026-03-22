import { describe, it, expect } from 'vitest';
import { groupReactions } from './groupReactions';

const reaction = (emoji: string, userId: string) => ({
  id: `r-${emoji}-${userId}`,
  message_id: 'msg1',
  user_id: userId,
  emoji,
  created_at: '2026-03-20T00:00:00Z',
});

describe('groupReactions', () => {
  it('returns empty array for no reactions', () => {
    expect(groupReactions([])).toEqual([]);
  });

  it('groups reactions by emoji', () => {
    const reactions = [reaction('👍', 'user1'), reaction('👍', 'user2'), reaction('❤️', 'user3')];
    const groups = groupReactions(reactions);
    expect(groups).toHaveLength(2);
    expect(groups[0]).toEqual({
      emoji: '👍',
      count: 2,
      userIds: ['user1', 'user2'],
      hasReacted: false,
    });
    expect(groups[1]).toEqual({
      emoji: '❤️',
      count: 1,
      userIds: ['user3'],
      hasReacted: false,
    });
  });

  it('marks hasReacted when current user has reacted', () => {
    const reactions = [reaction('👍', 'user1'), reaction('👍', 'me'), reaction('❤️', 'user1')];
    const groups = groupReactions(reactions, 'me');
    expect(groups[0].hasReacted).toBe(true);
    expect(groups[1].hasReacted).toBe(false);
  });

  it('marks hasReacted when current user is first reactor', () => {
    const reactions = [reaction('🎉', 'me')];
    const groups = groupReactions(reactions, 'me');
    expect(groups[0].hasReacted).toBe(true);
  });

  it('preserves insertion order of emojis', () => {
    const reactions = [reaction('❤️', 'user1'), reaction('👍', 'user2'), reaction('🎉', 'user3')];
    const groups = groupReactions(reactions);
    expect(groups.map((g) => g.emoji)).toEqual(['❤️', '👍', '🎉']);
  });

  it('handles undefined currentUserId', () => {
    const reactions = [reaction('👍', 'user1')];
    const groups = groupReactions(reactions, undefined);
    expect(groups[0].hasReacted).toBe(false);
  });

  it('collects all userIds per emoji', () => {
    const reactions = [reaction('👍', 'a'), reaction('👍', 'b'), reaction('👍', 'c')];
    const groups = groupReactions(reactions);
    expect(groups[0].userIds).toEqual(['a', 'b', 'c']);
    expect(groups[0].count).toBe(3);
  });
});
