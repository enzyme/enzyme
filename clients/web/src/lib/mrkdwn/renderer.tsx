import { useMemo } from 'react';
import { tv } from 'tailwind-variants';
import { parseMrkdwn, type MrkdwnSegment } from './parser';
import { UserMentionBadge, SpecialMentionBadge } from '../../components/message/MentionBadge';
import { ChannelMentionBadge } from '../../components/message/ChannelMentionBadge';
import type { WorkspaceMemberWithUser, ChannelWithMembership, CustomEmoji } from '@feather/api-client';
import { resolveStandardShortcode } from '../emoji';
import { CustomEmojiImg } from '../../components/ui/CustomEmojiImg';
import { isEmojiOnly } from './isEmojiOnly';

const styles = tv({
  slots: {
    code: [
      'bg-gray-100 dark:bg-gray-700',
      'px-1 py-0.5 rounded',
      'text-sm font-mono',
      'text-pink-600 dark:text-pink-400',
    ],
    codeBlock: [
      'block bg-gray-100 dark:bg-gray-800',
      'p-3 rounded-lg my-2',
      'overflow-x-auto',
      'text-sm font-mono',
      'whitespace-pre',
    ],
    blockquote: [
      'border-l-4 border-gray-300 dark:border-gray-600',
      'pl-3 my-1',
      'text-gray-600 dark:text-gray-400',
      'italic',
    ],
    list: [
      'my-1 pl-5',
    ],
    link: [
      'text-primary-600 dark:text-primary-400',
      'underline hover:no-underline',
    ],
  },
});

interface MrkdwnRendererProps {
  content: string;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  customEmojiMap?: Map<string, CustomEmoji>;
}

export function MrkdwnRenderer({ content, members = [], channels = [], customEmojiMap }: MrkdwnRendererProps) {
  const memberMap = useMemo(() => {
    return members.reduce((acc, member) => {
      acc[member.user_id] = member;
      return acc;
    }, {} as Record<string, WorkspaceMemberWithUser>);
  }, [members]);

  const segments = useMemo(() => parseMrkdwn(content), [content]);
  const emojiOnly = useMemo(() => isEmojiOnly(segments), [segments]);

  const rendered = renderSegments(segments, memberMap, channels, customEmojiMap, emojiOnly);

  if (emojiOnly) {
    return <span className="text-4xl leading-normal">{rendered}</span>;
  }

  return <>{rendered}</>;
}

function renderSegments(
  segments: MrkdwnSegment[],
  memberMap: Record<string, WorkspaceMemberWithUser>,
  channels: ChannelWithMembership[],
  customEmojiMap?: Map<string, CustomEmoji>,
  emojiOnly = false,
): React.ReactNode[] {
  const s = styles();

  return segments.map((segment, index) => {
    const key = `${segment.type}-${index}`;

    switch (segment.type) {
      case 'text':
        return <span key={key}>{segment.content}</span>;

      case 'bold':
        return <strong key={key}>{segment.content}</strong>;

      case 'italic':
        return <em key={key}>{segment.content}</em>;

      case 'strike':
        return <s key={key}>{segment.content}</s>;

      case 'code':
        return <code key={key} className={s.code()}>{segment.content}</code>;

      case 'code_block':
        return (
          <pre key={key} className={s.codeBlock()}>
            <code>{segment.content}</code>
          </pre>
        );

      case 'blockquote':
        return (
          <blockquote key={key} className={s.blockquote()}>
            {renderSegments(segment.segments, memberMap, channels, customEmojiMap)}
          </blockquote>
        );

      case 'bullet_list':
        return (
          <ul key={key} className={`${s.list()} list-disc`}>
            {segment.items.map((item, i) => (
              <li key={i}>{renderSegments(item, memberMap, channels, customEmojiMap)}</li>
            ))}
          </ul>
        );

      case 'ordered_list':
        return (
          <ol key={key} className={`${s.list()} list-decimal`}>
            {segment.items.map((item, i) => (
              <li key={i}>{renderSegments(item, memberMap, channels, customEmojiMap)}</li>
            ))}
          </ol>
        );

      case 'user_mention':
        return (
          <UserMentionBadge
            key={key}
            userId={segment.userId}
            member={memberMap[segment.userId]}
          />
        );

      case 'special_mention':
        return (
          <SpecialMentionBadge
            key={key}
            type={segment.mentionType}
          />
        );

      case 'channel_mention':
        return (
          <ChannelMentionBadge
            key={key}
            channelId={segment.channelId}
            channels={channels}
          />
        );

      case 'emoji_shortcode': {
        const standardEmoji = resolveStandardShortcode(segment.name);
        if (standardEmoji) {
          return <span key={key} title={`:${segment.name}:`}>{standardEmoji}</span>;
        }
        const customEmoji = customEmojiMap?.get(segment.name);
        if (customEmoji) {
          return <CustomEmojiImg key={key} name={customEmoji.name} url={customEmoji.url} size={emojiOnly ? 'xl' : 'sm'} />;
        }
        return <span key={key}>:{segment.name}:</span>;
      }

      case 'link':
        return (
          <a
            key={key}
            href={segment.url}
            target="_blank"
            rel="noopener noreferrer"
            className={s.link()}
          >
            {segment.text}
          </a>
        );

      case 'line_break':
        return <br key={key} />;

      default:
        return null;
    }
  });
}
