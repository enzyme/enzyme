import { forwardRef, useImperativeHandle, useState, useRef, useLayoutEffect } from 'react';
import { BellIcon } from '@heroicons/react/24/outline';
import { tv } from 'tailwind-variants';
import { Avatar } from '../../ui';
import type { MentionOption } from '../../../lib/mentions';

const styles = tv({
  slots: {
    container: [
      'z-50 bg-white dark:bg-gray-800',
      'border border-gray-200 dark:border-gray-700',
      'rounded-lg shadow-lg py-1 min-w-[200px] max-w-[300px] max-h-[240px] overflow-y-auto',
    ],
    item: [
      'w-full px-3 py-2 flex items-center gap-2 cursor-pointer',
      'hover:bg-gray-100 dark:hover:bg-gray-700',
      'outline-none',
    ],
    itemSelected: ['bg-gray-100 dark:bg-gray-700'],
    displayName: ['text-sm font-medium text-gray-900 dark:text-white truncate'],
    specialIcon: [
      'w-6 h-6 p-1 rounded-full bg-yellow-500 dark:bg-yellow-500 text-white dark:text-white',
    ],
    emptyState: ['px-3 py-2 text-sm text-gray-500 dark:text-gray-400'],
  },
});

export interface MentionSuggestionRef {
  onKeyDown: (props: { event: KeyboardEvent }) => boolean;
}

interface MentionSuggestionListProps {
  items: MentionOption[];
  command: (item: MentionOption) => void;
}

export const MentionSuggestionList = forwardRef<MentionSuggestionRef, MentionSuggestionListProps>(
  ({ items, command }, ref) => {
    // Use items.length as part of the initial state to handle resets
    // This works because React will reset state when items array reference changes
    const [selectedIndex, setSelectedIndex] = useState(0);
    const containerRef = useRef<HTMLDivElement>(null);
    const selectedRef = useRef<HTMLDivElement>(null);
    const s = styles();

    // Ensure selected index is within bounds
    const safeSelectedIndex = Math.min(selectedIndex, Math.max(0, items.length - 1));

    const selectItem = (index: number) => {
      const item = items[index];
      if (item) {
        command(item);
      }
    };

    const upHandler = () => {
      setSelectedIndex((prev) => (prev > 0 ? prev - 1 : items.length - 1));
    };

    const downHandler = () => {
      setSelectedIndex((prev) => (prev < items.length - 1 ? prev + 1 : 0));
    };

    const enterHandler = () => {
      selectItem(safeSelectedIndex);
    };

    // Scroll selected item into view
    useLayoutEffect(() => {
      if (selectedRef.current && containerRef.current) {
        const container = containerRef.current;
        const selected = selectedRef.current;

        const containerTop = container.scrollTop;
        const containerBottom = containerTop + container.clientHeight;
        const selectedTop = selected.offsetTop;
        const selectedBottom = selectedTop + selected.offsetHeight;

        if (selectedTop < containerTop) {
          container.scrollTop = selectedTop;
        } else if (selectedBottom > containerBottom) {
          container.scrollTop = selectedBottom - container.clientHeight;
        }
      }
    }, [safeSelectedIndex]);

    useImperativeHandle(ref, () => ({
      onKeyDown: ({ event }) => {
        if (event.key === 'ArrowUp') {
          upHandler();
          return true;
        }

        if (event.key === 'ArrowDown') {
          downHandler();
          return true;
        }

        if (event.key === 'Enter' || event.key === 'Tab') {
          enterHandler();
          return true;
        }

        return false;
      },
    }));

    if (items.length === 0) {
      return (
        <div ref={containerRef} className={s.container()}>
          <div className={s.emptyState()}>No matches found</div>
        </div>
      );
    }

    return (
      <div ref={containerRef} className={s.container()}>
        {items.map((item, index) => (
          <div
            key={`${item.type}-${item.id}`}
            ref={index === safeSelectedIndex ? selectedRef : undefined}
            className={`${s.item()} ${index === safeSelectedIndex ? s.itemSelected() : ''}`}
            onClick={() => selectItem(index)}
            onMouseEnter={() => setSelectedIndex(index)}
            role="option"
            aria-selected={index === safeSelectedIndex}
          >
            {item.type === 'user' ? (
              <Avatar
                src={item.avatarUrl}
                gravatarSrc={item.gravatarUrl}
                name={item.displayName}
                id={item.id}
                size="sm"
              />
            ) : (
              <div className={s.specialIcon()}>
                <BellIcon className="h-4 w-4" />
              </div>
            )}
            <span className={s.displayName()}>
              {item.type === 'special' ? `@${item.displayName}` : item.displayName}
            </span>
          </div>
        ))}
      </div>
    );
  },
);

MentionSuggestionList.displayName = 'MentionSuggestionList';
