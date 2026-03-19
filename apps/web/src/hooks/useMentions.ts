import { useMemo } from 'react';
import { useWorkspaceMembers } from './useWorkspaces';
import {
  parseMentionTrigger,
  SPECIAL_MENTIONS,
  type MentionOption,
  type MentionTrigger,
} from '@enzyme/shared';

interface UseMentionsResult {
  trigger: MentionTrigger | null;
  options: MentionOption[];
}

export function useMentions(
  workspaceId: string | undefined,
  content: string,
  cursorPosition: number,
): UseMentionsResult {
  const { data: membersData } = useWorkspaceMembers(workspaceId);

  const trigger = useMemo(() => {
    return parseMentionTrigger(content, cursorPosition);
  }, [content, cursorPosition]);

  const options = useMemo(() => {
    if (!trigger?.isActive) {
      return [];
    }

    const query = trigger.query.toLowerCase();

    // Convert members to MentionOptions
    const memberOptions: MentionOption[] = (membersData?.members ?? []).map((member) => ({
      type: 'user' as const,
      id: member.user_id,
      displayName: member.display_name,
      avatarUrl: member.avatar_url,
      gravatarUrl: member.gravatar_url,
    }));

    // Combine members and special mentions
    const allOptions = [...memberOptions, ...SPECIAL_MENTIONS];

    // Filter by query
    const filtered = allOptions.filter((option) =>
      option.displayName.toLowerCase().includes(query),
    );

    // Sort: exact matches first, then by display name
    return filtered.sort((a, b) => {
      const aStartsWith = a.displayName.toLowerCase().startsWith(query);
      const bStartsWith = b.displayName.toLowerCase().startsWith(query);

      if (aStartsWith && !bStartsWith) return -1;
      if (!aStartsWith && bStartsWith) return 1;

      return a.displayName.localeCompare(b.displayName);
    });
  }, [trigger, membersData?.members]);

  return { trigger, options };
}
