import { useState, useRef, useLayoutEffect, useCallback, type ReactNode } from 'react';
import { cn } from '../../lib/utils';

const COLLAPSED_HEIGHT = 600;
const TRANSITION_MS = 300;

interface CollapsibleMessageProps {
  children: ReactNode;
}

export function CollapsibleMessage({ children }: CollapsibleMessageProps) {
  const contentRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [isOverflowing, setIsOverflowing] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);
  const [isAnimating, setIsAnimating] = useState(false);
  const [animMaxHeight, setAnimMaxHeight] = useState<number | undefined>(undefined);

  useLayoutEffect(() => {
    const el = contentRef.current;
    if (!el) return;

    const observer = new ResizeObserver(() => {
      setIsOverflowing(el.scrollHeight > COLLAPSED_HEIGHT);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const handleExpand = useCallback(() => {
    const el = contentRef.current;
    if (!el) return;
    setAnimMaxHeight(el.scrollHeight);
    setIsAnimating(true);
    setIsExpanded(true);
  }, []);

  const collapsingRef = useRef(false);

  const handleCollapse = useCallback(() => {
    const el = contentRef.current;
    if (!el) return;
    collapsingRef.current = true;
    // Pin to current height first (no transition yet)
    setAnimMaxHeight(el.scrollHeight);
    setIsAnimating(true);
    // Next frame: transition to collapsed height
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        setAnimMaxHeight(COLLAPSED_HEIGHT);
        setIsExpanded(false);
      });
    });
  }, []);

  const handleTransitionEnd = useCallback((e: React.TransitionEvent) => {
    if (e.propertyName !== 'max-height') return;
    setIsAnimating(false);
    setAnimMaxHeight(undefined);
    // After collapse animation, scroll into view if top is above viewport
    if (collapsingRef.current) {
      collapsingRef.current = false;
      const wrapper = wrapperRef.current;
      if (wrapper) {
        const rect = wrapper.getBoundingClientRect();
        if (rect.top < 0) {
          wrapper.scrollIntoView({ block: 'start', behavior: 'smooth' });
        }
      }
    }
  }, []);

  // Compute the style for the content container
  let contentStyle: React.CSSProperties | undefined;
  if (isOverflowing) {
    if (isAnimating) {
      contentStyle = {
        maxHeight: animMaxHeight,
        overflow: 'hidden',
        transition: `max-height ${TRANSITION_MS}ms ease-in-out`,
      };
    } else if (!isExpanded) {
      contentStyle = { maxHeight: COLLAPSED_HEIGHT, overflow: 'hidden' };
    }
  }

  const showGradient = isOverflowing && !isExpanded && !isAnimating;

  return (
    <div ref={wrapperRef}>
      <div
        ref={contentRef}
        className="relative"
        style={contentStyle}
        onTransitionEnd={handleTransitionEnd}
      >
        {children}
        {isOverflowing && !isExpanded && (
          <div
            className={cn(
              'pointer-events-none absolute inset-x-0 bottom-0 h-20 bg-gradient-to-t from-white to-transparent transition-opacity duration-200 group-hover:from-gray-100 dark:from-gray-900 dark:group-hover:from-gray-800',
              showGradient ? 'opacity-100' : 'opacity-0',
            )}
          />
        )}
      </div>
      {isOverflowing && (
        <button
          type="button"
          onClick={isExpanded ? handleCollapse : handleExpand}
          className="flex w-full cursor-pointer items-center justify-center gap-1 py-2 text-xs font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
        >
          {isExpanded ? 'Show less' : 'Show more'}
          <svg
            className={cn(
              'h-3 w-3 transition-transform duration-200',
              isExpanded ? '-rotate-90' : 'rotate-90',
            )}
            viewBox="0 0 12 12"
            fill="currentColor"
          >
            <path d="M4 2 L8 6 L4 10 Z" />
          </svg>
        </button>
      )}
    </div>
  );
}
