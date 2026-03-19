import { UnstyledButton, Tooltip, CustomEmojiImg } from '../ui';
import { cn } from '../../lib/utils';
import { resolveStandardShortcode } from '@enzyme/shared';
import type { ReactionGroup } from './reactionUtils';
import type { CustomEmoji } from '@enzyme/api-client';

/**
 * Render an emoji string which may be:
 * 1. A shortcode like `:thumbsup:` (standard or custom)
 * 2. A raw Unicode character (legacy)
 */
export function EmojiDisplay({
  emoji,
  customEmojiMap,
  size = 'sm',
}: {
  emoji: string;
  customEmojiMap?: Map<string, CustomEmoji>;
  size?: 'sm' | 'md';
}) {
  // Check if it's a shortcode format :name:
  const shortcodeMatch = emoji.match(/^:([a-zA-Z0-9_+-]+):$/);
  if (shortcodeMatch) {
    const name = shortcodeMatch[1];
    // Try standard emoji
    const standard = resolveStandardShortcode(name);
    if (standard) return <>{standard}</>;
    // Try custom emoji
    const custom = customEmojiMap?.get(name);
    if (custom) return <CustomEmojiImg name={custom.name} url={custom.url} size={size} />;
    // Fallback: render the shortcode as text
    return <>{emoji}</>;
  }
  // Raw Unicode (legacy)
  return <>{emoji}</>;
}

interface ReactionsDisplayProps {
  reactions: ReactionGroup[];
  memberNames: Record<string, string>;
  onReactionClick: (emoji: string, hasOwn: boolean) => void;
  customEmojiMap?: Map<string, CustomEmoji>;
}

export function ReactionsDisplay({
  reactions,
  memberNames,
  onReactionClick,
  customEmojiMap,
}: ReactionsDisplayProps) {
  if (reactions.length === 0) {
    return null;
  }

  return (
    <div className="mt-1 flex flex-wrap gap-1">
      {reactions.map(({ emoji, count, userIds, hasOwn }) => {
        const userNames = userIds.map((id) => memberNames[id] || 'Unknown').join(', ');
        return (
          <Tooltip key={emoji} content={userNames}>
            <UnstyledButton
              onPress={() => onReactionClick(emoji, hasOwn)}
              className={cn(
                'inline-flex cursor-pointer items-center gap-1 rounded-full border px-2 py-0.5 text-sm transition-colors',
                hasOwn
                  ? 'border-blue-300 bg-blue-100 dark:border-blue-700 dark:bg-blue-900'
                  : 'border-gray-200 bg-gray-100 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:hover:bg-gray-600',
              )}
            >
              <span>
                <EmojiDisplay emoji={emoji} customEmojiMap={customEmojiMap} size="md" />
              </span>
              <span className="text-xs text-gray-600 dark:text-gray-300">{count}</span>
            </UnstyledButton>
          </Tooltip>
        );
      })}
    </div>
  );
}
