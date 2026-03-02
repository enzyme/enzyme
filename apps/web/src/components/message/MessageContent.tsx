import type {
  WorkspaceMemberWithUser,
  ChannelWithMembership,
  CustomEmoji,
} from '@enzyme/api-client';
import { MrkdwnRenderer } from '../../lib/mrkdwn';

interface MessageContentProps {
  content: string;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  customEmojiMap?: Map<string, CustomEmoji>;
}

export function EditedBadge({ inline }: { inline?: boolean }) {
  return (
    <span className="text-sm text-gray-400 dark:text-gray-500">{inline ? ' ' : ''}(edited)</span>
  );
}

export function MessageContent({
  content,
  members = [],
  channels = [],
  customEmojiMap,
}: MessageContentProps) {
  return (
    <MrkdwnRenderer
      content={content}
      members={members}
      channels={channels}
      customEmojiMap={customEmojiMap}
    />
  );
}
