import { useRef, useCallback, useEffect } from 'react';

interface UseAutoScrollOptions {
  threshold?: number; // Distance from bottom to consider "at bottom"
}

export function useAutoScroll<T extends HTMLElement>(
  options: UseAutoScrollOptions = {}
) {
  const { threshold = 100 } = options;
  const containerRef = useRef<T>(null);
  const shouldScrollRef = useRef(true);
  const prevScrollHeightRef = useRef(0);

  const checkIfAtBottom = useCallback(() => {
    const container = containerRef.current;
    if (!container) return true;

    const { scrollTop, scrollHeight, clientHeight } = container;
    return scrollHeight - scrollTop - clientHeight < threshold;
  }, [threshold]);

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    const container = containerRef.current;
    if (!container) return;

    container.scrollTo({
      top: container.scrollHeight,
      behavior,
    });
  }, []);

  // Call this before prepending older messages to preserve scroll position
  const preserveScrollPosition = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;

    prevScrollHeightRef.current = container.scrollHeight;
  }, []);

  // Call this after prepending older messages
  const restoreScrollPosition = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;

    const newScrollHeight = container.scrollHeight;
    const scrollDiff = newScrollHeight - prevScrollHeightRef.current;
    container.scrollTop += scrollDiff;
  }, []);

  // Track scroll position to determine if we should auto-scroll
  const handleScroll = useCallback(() => {
    shouldScrollRef.current = checkIfAtBottom();
  }, [checkIfAtBottom]);

  // Auto-scroll when new messages arrive (if user was at bottom)
  const scrollOnNewMessage = useCallback(() => {
    if (shouldScrollRef.current) {
      scrollToBottom('smooth');
    }
  }, [scrollToBottom]);

  // Initial scroll to bottom
  useEffect(() => {
    scrollToBottom('instant');
  }, [scrollToBottom]);

  return {
    containerRef,
    scrollToBottom,
    scrollOnNewMessage,
    preserveScrollPosition,
    restoreScrollPosition,
    handleScroll,
    isAtBottom: checkIfAtBottom,
  };
}
