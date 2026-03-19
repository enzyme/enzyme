import { useRef, useEffect, useLayoutEffect, useState } from 'react';
import { BellIcon } from '@heroicons/react/24/outline';
import { tv } from 'tailwind-variants';
import { Avatar } from './Avatar';
import type { MentionOption } from '@enzyme/shared';

const mentionPopover = tv({
  slots: {
    container: [
      'absolute z-50 bg-white dark:bg-gray-800',
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
      'w-6 h-6 p-1 rounded-full bg-yellow-100 dark:bg-yellow-900 text-yellow-600 dark:text-yellow-400',
    ],
    emptyState: ['px-3 py-2 text-sm text-gray-500 dark:text-gray-400'],
  },
});

interface MentionPopoverProps {
  options: MentionOption[];
  selectedIndex: number;
  onSelect: (option: MentionOption) => void;
  leftOffset: number;
}

export function MentionPopover({
  options,
  selectedIndex,
  onSelect,
  leftOffset,
}: MentionPopoverProps) {
  const styles = mentionPopover();
  const containerRef = useRef<HTMLDivElement>(null);
  const selectedRef = useRef<HTMLDivElement>(null);
  const [adjustedLeft, setAdjustedLeft] = useState(leftOffset);

  // Adjust position to stay within bounds
  useLayoutEffect(() => {
    if (containerRef.current) {
      const popover = containerRef.current;
      const parent = popover.offsetParent as HTMLElement | null;

      if (parent) {
        const parentWidth = parent.clientWidth;
        const popoverWidth = popover.offsetWidth;
        const maxLeft = parentWidth - popoverWidth;

        // Clamp left position between 0 and maxLeft
        setAdjustedLeft(Math.max(0, Math.min(leftOffset, maxLeft)));
      } else {
        setAdjustedLeft(Math.max(0, leftOffset));
      }
    }
  }, [leftOffset]);

  // Scroll selected item into view
  useEffect(() => {
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
  }, [selectedIndex]);

  if (options.length === 0) {
    return (
      <div
        className={styles.container()}
        style={{
          bottom: '100%',
          left: adjustedLeft,
          marginBottom: '4px',
        }}
      >
        <div className={styles.emptyState()}>No matches found</div>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className={styles.container()}
      style={{
        bottom: '100%',
        left: adjustedLeft,
        marginBottom: '4px',
      }}
    >
      {options.map((option, index) => (
        <div
          key={`${option.type}-${option.id}`}
          ref={index === selectedIndex ? selectedRef : undefined}
          className={`${styles.item()} ${index === selectedIndex ? styles.itemSelected() : ''}`}
          onClick={() => onSelect(option)}
          onMouseDown={(e) => e.preventDefault()} // Prevent blur on textarea
          role="option"
          aria-selected={index === selectedIndex}
        >
          {option.type === 'user' ? (
            <Avatar
              src={option.avatarUrl}
              gravatarSrc={option.gravatarUrl}
              name={option.displayName}
              size="sm"
            />
          ) : (
            <div className={styles.specialIcon()}>
              <BellIcon className="h-4 w-4" />
            </div>
          )}
          <span className={styles.displayName()}>
            {option.type === 'special' ? `@${option.displayName}` : option.displayName}
          </span>
        </div>
      ))}
    </div>
  );
}
