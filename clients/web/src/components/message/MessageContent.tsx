import { useMemo } from 'react';
import { parseStoredMentions } from '../../lib/mentions';
import { UserMentionBadge, SpecialMentionBadge } from './MentionBadge';
import type { WorkspaceMemberWithUser } from '@feather/api-client';

interface MessageContentProps {
  content: string;
  members?: WorkspaceMemberWithUser[];
}

export function MessageContent({ content, members = [] }: MessageContentProps) {
  // Create a lookup map from user ID to member
  const memberMap = useMemo(() => {
    return members.reduce((acc, member) => {
      acc[member.user_id] = member;
      return acc;
    }, {} as Record<string, WorkspaceMemberWithUser>);
  }, [members]);

  const segments = useMemo(() => parseStoredMentions(content), [content]);

  return (
    <>
      {segments.map((segment, index) => {
        if (segment.type === 'text') {
          return <span key={index}>{segment.content}</span>;
        }

        if (segment.type === 'user_mention') {
          return (
            <UserMentionBadge
              key={index}
              userId={segment.content}
              member={memberMap[segment.content]}
            />
          );
        }

        if (segment.type === 'special_mention') {
          return (
            <SpecialMentionBadge
              key={index}
              type={segment.content}
            />
          );
        }

        return null;
      })}
    </>
  );
}
