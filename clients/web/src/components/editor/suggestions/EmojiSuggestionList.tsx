import {
  forwardRef,
  useImperativeHandle,
  useState,
  useRef,
  useLayoutEffect,
} from 'react';
import { tv } from 'tailwind-variants';
import { CustomEmojiImg } from '../../ui/CustomEmojiImg';

const styles = tv({
  slots: {
    container: [
      'z-50 bg-white dark:bg-gray-800',
      'border border-gray-200 dark:border-gray-700',
      'rounded-lg shadow-lg py-1 min-w-[200px] max-w-[280px] max-h-[240px] overflow-y-auto',
    ],
    item: [
      'w-full px-3 py-1.5 flex items-center gap-2 cursor-pointer',
      'hover:bg-gray-100 dark:hover:bg-gray-700',
      'outline-none',
    ],
    itemSelected: [
      'bg-gray-100 dark:bg-gray-700',
    ],
    emoji: 'text-lg',
    shortcode: 'text-sm text-gray-600 dark:text-gray-400',
    emptyState: [
      'px-3 py-2 text-sm text-gray-500 dark:text-gray-400',
    ],
  },
});

export interface EmojiOption {
  shortcode: string;
  emoji: string;
  isCustom?: boolean;
  imageUrl?: string;
}

export interface EmojiSuggestionRef {
  onKeyDown: (props: { event: KeyboardEvent }) => boolean;
}

interface EmojiSuggestionListProps {
  items: EmojiOption[];
  command: (item: EmojiOption) => void;
}

export const EmojiSuggestionList = forwardRef<EmojiSuggestionRef, EmojiSuggestionListProps>(
  ({ items, command }, ref) => {
    const [selectedIndex, setSelectedIndex] = useState(0);
    const containerRef = useRef<HTMLDivElement>(null);
    const selectedRef = useRef<HTMLDivElement>(null);
    const s = styles();

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
          <div className={s.emptyState()}>No emoji found</div>
        </div>
      );
    }

    return (
      <div ref={containerRef} className={s.container()}>
        {items.map((item, index) => (
          <div
            key={item.shortcode}
            ref={index === safeSelectedIndex ? selectedRef : undefined}
            className={`${s.item()} ${index === safeSelectedIndex ? s.itemSelected() : ''}`}
            onClick={() => selectItem(index)}
            onMouseEnter={() => setSelectedIndex(index)}
            role="option"
            aria-selected={index === safeSelectedIndex}
          >
            <span className={s.emoji()}>
              {item.isCustom && item.imageUrl
                ? <CustomEmojiImg name={item.shortcode} url={item.imageUrl} size="md" />
                : item.emoji}
            </span>
            <span className={s.shortcode()}>:{item.shortcode}:</span>
          </div>
        ))}
      </div>
    );
  }
);

EmojiSuggestionList.displayName = 'EmojiSuggestionList';
