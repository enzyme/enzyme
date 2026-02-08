import type { WorkspaceMemberWithUser, ChannelWithMembership, CustomEmoji } from '@feather/api-client';
import { MrkdwnRenderer } from '../../lib/mrkdwn';

interface MessageContentProps {
  content: string;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  customEmojiMap?: Map<string, CustomEmoji>;
}

export function MessageContent({ content, members = [], channels = [], customEmojiMap }: MessageContentProps) {
  return <MrkdwnRenderer content={content} members={members} channels={channels} customEmojiMap={customEmojiMap} />;
}
