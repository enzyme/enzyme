import type { WorkspaceMemberWithUser, ChannelWithMembership } from '@feather/api-client';
import { MrkdwnRenderer } from '../../lib/mrkdwn';

interface MessageContentProps {
  content: string;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
}

export function MessageContent({ content, members = [], channels = [] }: MessageContentProps) {
  return <MrkdwnRenderer content={content} members={members} channels={channels} />;
}
