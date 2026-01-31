import { cn } from '../../lib/utils';

const COMMON_EMOJIS = [
  '👍', '👎', '❤️', '😂', '😮', '😢', '😡', '🎉',
  '🔥', '👀', '✅', '❌', '🚀', '💯', '🙏', '👏',
];

interface ReactionPickerProps {
  onSelect: (emoji: string) => void;
}

export function ReactionPicker({ onSelect }: ReactionPickerProps) {
  return (
    <div className={cn(
      'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700',
      'rounded-lg shadow-lg p-2'
    )}>
      <div className="grid grid-cols-8 gap-1">
        {COMMON_EMOJIS.map((emoji) => (
          <button
            key={emoji}
            onClick={() => onSelect(emoji)}
            className={cn(
              'w-8 h-8 flex items-center justify-center rounded',
              'hover:bg-gray-100 dark:hover:bg-gray-700',
              'text-lg transition-colors'
            )}
          >
            {emoji}
          </button>
        ))}
      </div>
    </div>
  );
}
