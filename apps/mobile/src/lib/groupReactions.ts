import type { Reaction } from '@enzyme/api-client';

export interface ReactionGroup {
  emoji: string;
  count: number;
  userIds: string[];
  hasReacted: boolean;
}

export function groupReactions(reactions: Reaction[], currentUserId?: string): ReactionGroup[] {
  const groups = new Map<string, ReactionGroup>();

  for (const r of reactions) {
    const existing = groups.get(r.emoji);
    if (existing) {
      existing.count++;
      existing.userIds.push(r.user_id);
      if (r.user_id === currentUserId) existing.hasReacted = true;
    } else {
      groups.set(r.emoji, {
        emoji: r.emoji,
        count: 1,
        userIds: [r.user_id],
        hasReacted: r.user_id === currentUserId,
      });
    }
  }

  return Array.from(groups.values());
}
