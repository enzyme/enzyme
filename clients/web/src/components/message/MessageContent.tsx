import type { WorkspaceMemberWithUser } from '@feather/api-client';
import { MrkdwnRenderer } from '../../lib/mrkdwn';

interface MessageContentProps {
  content: string;
  members?: WorkspaceMemberWithUser[];
}

export function MessageContent({ content, members = [] }: MessageContentProps) {
  return <MrkdwnRenderer content={content} members={members} />;
}
