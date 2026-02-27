import React, { useEffect, useLayoutEffect, useRef, useCallback, useMemo, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { useVirtualizer } from '@tanstack/react-virtual';
import { useAuth, useChannels } from '../../hooks';
import { useVirtualMessages } from '../../hooks/useVirtualMessages';
import { usePrewarmSignedUrls } from '../../hooks/usePrewarmSignedUrls';
import { buildVirtualItems, type VirtualItem } from './virtualItems';
import { MessageItem } from './MessageItem';
import { SystemMessage } from './SystemMessage';
import { JumpToLatestButton } from './JumpToLatestButton';
import { MessageSkeleton } from '../ui';
import { formatDate } from '../../lib/utils';
import type { ChannelWithMembership } from '@enzyme/api-client';

interface MessageListProps {
  channelId: string;
  lastReadMessageId?: string;
  unreadCount?: number;
  onAtBottomChange?: (isAtBottom: boolean) => void;
}

export function MessageList({
  channelId,
  lastReadMessageId,
  unreadCount = 0,
  onAtBottomChange,
}: MessageListProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const {
    data,
    isLoading,
    isFetchingNextPage,
    isFetchingPreviousPage,
    hasNextPage,
    hasPreviousPage,
    fetchNextPage,
    fetchPreviousPage,
    isDetached,
    jumpToMessage,
    jumpToLatest,
    onReachBottom,
  } = useVirtualMessages(channelId);
  const { data: channelsData } = useChannels(workspaceId);
  const { workspaces } = useAuth();
  const workspaceRole = workspaces?.find((w) => w.id === workspaceId)?.role;
  const isAdmin = workspaceRole === 'owner' || workspaceRole === 'admin';

  usePrewarmSignedUrls(data?.pages);

  // Read search params early so the initial-scroll layout effect can skip scroll-to-bottom
  // when a ?msg= deep link is present (the ?msg= handler will scroll to the target instead).
  // Skip ?msg= when ?thread= is present — ThreadPanel handles thread reply highlights.
  const [searchParams, setSearchParams] = useSearchParams();
  const highlightedMsgId = searchParams.get('thread') ? null : searchParams.get('msg');
  const highlightedMsgIdRef = useRef(highlightedMsgId);
  highlightedMsgIdRef.current = highlightedMsgId;

  const containerRef = useRef<HTMLDivElement>(null);
  const topSentinelRef = useRef<HTMLDivElement>(null);
  const bottomSentinelRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef(true);
  const lastBottomMessageIdRef = useRef<string | null>(null);
  const scrollAnchorRef = useRef<{ itemKey: string; offsetFromContainerTop: number } | null>(null);
  const programmaticScrollRef = useRef(true); // Start true — initial scroll pending
  const [showJumpButton, setShowJumpButton] = useState(false);
  const [newMessageCount, setNewMessageCount] = useState(0);

  // Refs to avoid stale closures in IntersectionObserver callbacks
  const isFetchingNextRef = useRef(false);
  const hasNextPageRef = useRef(false);
  const isFetchingPrevRef = useRef(false);
  const hasPreviousPageRef = useRef(false);
  isFetchingNextRef.current = isFetchingNextPage;
  hasNextPageRef.current = hasNextPage ?? false;
  isFetchingPrevRef.current = isFetchingPreviousPage;
  hasPreviousPageRef.current = hasPreviousPage ?? false;
  // Flatten and sort messages (pages come newest-first within each page, sort by ID for correctness)
  const allMessages = useMemo(() => {
    if (!data?.pages) return [];
    const flat = data.pages.flatMap((page) => page.messages);
    // Sort by ID (ULIDs are lexicographically ordered by time) and deduplicate
    const seen = new Set<string>();
    return flat
      .sort((a, b) => a.id.localeCompare(b.id))
      .filter((m) => {
        if (seen.has(m.id)) return false;
        seen.add(m.id);
        return true;
      });
  }, [data?.pages]);

  // Build virtual items (date separators, messages, unread divider)
  const virtualItems = useMemo(
    () => buildVirtualItems(allMessages, lastReadMessageId, unreadCount),
    [allMessages, lastReadMessageId, unreadCount],
  );

  // Use refs for virtualItems in estimateSize/getItemKey to keep function identity
  // stable. If estimateSize identity changes, TanStack Virtual clears its entire
  // measurement cache — which resets all item sizes to estimates and causes the
  // total virtual size to collapse, triggering cascading scroll corrections.
  const virtualItemsRef = useRef(virtualItems);
  virtualItemsRef.current = virtualItems;

  const estimateSize = useCallback((index: number) => {
    const item = virtualItemsRef.current[index];
    if (!item) return 72;
    if (item.type === 'date-separator') return 44;
    if (item.type === 'unread-divider') return 40;
    if (item.type === 'message' && item.message.link_preview) return 150;
    return 72; // average message height
  }, []);

  const getItemKey = useCallback(
    (index: number) => virtualItemsRef.current[index]?.key ?? String(index),
    [],
  );

  const virtualizer = useVirtualizer({
    count: virtualItems.length,
    getScrollElement: () => containerRef.current,
    estimateSize,
    getItemKey,
    overscan: 10,
    measureElement: (el) => el.getBoundingClientRect().height,
  });

  // Disable TanStack Virtual's scroll correction mechanism. When a rendered item's
  // measured size differs from its estimate, the virtualizer normally calls
  // scrollTo(offset + delta) to keep viewport content stable. But this fires from
  // ResizeObserver in the same frame as our rAF settle loops, using a stale scroll
  // offset and overriding our scrollTop assignments — causing jitter.
  // We manage scroll position ourselves via settle loops and anchor restoration.
  virtualizer.shouldAdjustScrollPositionOnItemSizeChange = () => false;

  // Synchronously correct element sizes after DOM commits. When content changes
  // (e.g. a link preview is dismissed via optimistic update), the DOM updates
  // immediately but the virtualizer's ResizeObserver notification is async — the
  // position recalculation paints one frame late, leaving a visible layout gap.
  // This effect runs in useLayoutEffect (before paint), collects all size
  // discrepancies first, then calls resizeItem for each — batching avoids
  // multiple intermediate notify/re-render cycles.
  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const vItems = virtualizer.getVirtualItems();
    const pending: Array<{ index: number; size: number }> = [];
    for (const vItem of vItems) {
      const el = container.querySelector(`[data-index="${vItem.index}"]`) as HTMLElement;
      if (el) {
        const actualHeight = el.getBoundingClientRect().height;
        if (Math.abs(actualHeight - vItem.size) > 0.5) {
          pending.push({ index: vItem.index, size: actualHeight });
        }
      }
    }
    for (const { index, size } of pending) {
      virtualizer.resizeItem(index, size);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only re-check when message data changes
  }, [virtualItems]);

  // Track new messages arriving while scrolled up by comparing against the
  // newest message ID from when the user was last at the bottom. This avoids
  // counting older messages loaded via infinite scroll as "new".
  useEffect(() => {
    if (allMessages.length === 0) return;
    const newestId = allMessages[allMessages.length - 1].id;

    if (isAtBottomRef.current) {
      lastBottomMessageIdRef.current = newestId;
      setNewMessageCount(0);
    } else if (lastBottomMessageIdRef.current) {
      const idx = allMessages.findIndex((m) => m.id > lastBottomMessageIdRef.current!);
      setNewMessageCount(idx === -1 ? 0 : allMessages.length - idx);
    } else {
      lastBottomMessageIdRef.current = newestId;
    }
  }, [allMessages]);

  // Handle scroll events — only tracks at-bottom state and jump button
  const allMessagesRef = useRef(allMessages);
  allMessagesRef.current = allMessages;

  const handleScroll = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    const atBottom = scrollHeight - scrollTop - clientHeight < 100;

    if (atBottom !== isAtBottomRef.current) {
      isAtBottomRef.current = atBottom;
      onAtBottomChange?.(atBottom);
      setShowJumpButton(!atBottom);
    }

    if (atBottom) {
      const msgs = allMessagesRef.current;
      if (msgs.length > 0) {
        lastBottomMessageIdRef.current = msgs[msgs.length - 1].id;
      }
      setNewMessageCount(0);
      onReachBottom();
    }
  }, [onAtBottomChange, onReachBottom]);

  // Save a scroll anchor before fetching so we can restore position after maxPages
  // evicts items from the opposite end. Uses virtualizer data (not DOM queries) so
  // it works even if the anchor element isn't rendered after re-render.
  const virtualizerRef = useRef(virtualizer);
  virtualizerRef.current = virtualizer;

  const saveScrollAnchor = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;
    const scrollTop = container.scrollTop;
    const vItems = virtualizer.getVirtualItems();
    // Find the first virtual item whose bottom edge is past the viewport top
    for (const vItem of vItems) {
      if (vItem.end > scrollTop) {
        const itemData = virtualItemsRef.current[vItem.index];
        if (itemData) {
          scrollAnchorRef.current = {
            itemKey: itemData.key,
            offsetFromContainerTop: vItem.start - scrollTop,
          };
          console.debug(
            `[scroll:saveAnchor] ${JSON.stringify({ itemKey: itemData.key, offset: vItem.start - scrollTop, scrollTop })}`,
          );
        }
        return;
      }
    }
  }, [virtualizer]);

  // IntersectionObserver: load older messages when top sentinel is near viewport
  useEffect(() => {
    const container = containerRef.current;
    const sentinel = topSentinelRef.current;
    if (!container || !sentinel) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        console.debug(
          `[scroll:observer] ${JSON.stringify({ isIntersecting: entry.isIntersecting, hasNext: hasNextPageRef.current, isFetching: isFetchingNextRef.current, blocked: programmaticScrollRef.current, scrollTop: container.scrollTop })}`,
        );
        if (
          entry.isIntersecting &&
          hasNextPageRef.current &&
          !isFetchingNextRef.current &&
          !programmaticScrollRef.current
        ) {
          saveScrollAnchor();
          fetchNextPage();
        }
      },
      { root: container, rootMargin: '500px 0px 0px 0px' },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [channelId, fetchNextPage, isLoading, saveScrollAnchor]);

  // IntersectionObserver: load newer messages when bottom sentinel is near viewport
  useEffect(() => {
    const container = containerRef.current;
    const sentinel = bottomSentinelRef.current;
    if (!container || !sentinel) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && hasPreviousPageRef.current && !isFetchingPrevRef.current) {
          saveScrollAnchor();
          fetchPreviousPage();
        }
      },
      { root: container, rootMargin: '0px 0px 500px 0px' },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [channelId, fetchPreviousPage, isLoading, saveScrollAnchor]);

  // Post-fetch re-check: if sentinel is still visible after a fetch completes, fetch again
  const wasFetchingNextRef = useRef(false);
  useEffect(() => {
    if (wasFetchingNextRef.current && !isFetchingNextPage) {
      const container = containerRef.current;
      const sentinel = topSentinelRef.current;
      if (container && sentinel && hasNextPage) {
        const containerRect = container.getBoundingClientRect();
        const sentinelRect = sentinel.getBoundingClientRect();
        const sentinelVisible = sentinelRect.bottom > containerRect.top - 500;
        console.debug(
          `[scroll:postFetchRecheck] ${JSON.stringify({ blocked: programmaticScrollRef.current, sentinelVisible })}`,
        );
        if (sentinelVisible && !programmaticScrollRef.current) {
          saveScrollAnchor();
          fetchNextPage();
        }
      }
    }
    wasFetchingNextRef.current = isFetchingNextPage;
  }, [isFetchingNextPage, hasNextPage, fetchNextPage, saveScrollAnchor]);

  const wasFetchingPrevRef = useRef(false);
  useEffect(() => {
    if (wasFetchingPrevRef.current && !isFetchingPreviousPage) {
      const container = containerRef.current;
      const sentinel = bottomSentinelRef.current;
      if (container && sentinel && hasPreviousPage) {
        const containerRect = container.getBoundingClientRect();
        const sentinelRect = sentinel.getBoundingClientRect();
        if (sentinelRect.top < containerRect.bottom + 500) {
          saveScrollAnchor();
          fetchPreviousPage();
        }
      }
    }
    wasFetchingPrevRef.current = isFetchingPreviousPage;
  }, [isFetchingPreviousPage, hasPreviousPage, fetchPreviousPage, saveScrollAnchor]);

  // Preserve scroll position when messages are loaded and maxPages evicts from the opposite end.
  // Uses a scroll anchor: before each fetch, we save a visible item's key and its viewport offset.
  // After re-render, we find that item's new index and use getOffsetForIndex (read-only) + direct
  // scrollTop assignment to restore position. We avoid scrollToIndex because its persistent retry
  // loop "undoes" the viewport-relative offset adjustment, causing visible jitter.
  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container || !scrollAnchorRef.current) return;

    const { itemKey, offsetFromContainerTop } = scrollAnchorRef.current;
    const newIndex = virtualItems.findIndex((item) => item.key === itemKey);
    scrollAnchorRef.current = null;

    if (newIndex !== -1) {
      console.debug(
        `[scroll:restoreAnchor] ${JSON.stringify({ itemKey, newIndex, offset: offsetFromContainerTop })}`,
      );
      // Use getOffsetForIndex (read-only) + direct scrollTop assignment.
      // scrollToIndex has a persistent retry loop that "undoes" the offset adjustment.
      const offsetInfo = virtualizerRef.current.getOffsetForIndex(newIndex, 'start');
      if (offsetInfo) {
        container.scrollTop = offsetInfo[0] + offsetFromContainerTop;
      }
    }

    // Second pass after items near the restored position are measured
    requestAnimationFrame(() => {
      if (newIndex !== -1) {
        const offsetInfo = virtualizerRef.current.getOffsetForIndex(newIndex, 'start');
        if (offsetInfo) {
          container.scrollTop = offsetInfo[0] + offsetFromContainerTop;
        }
      }
    });
  }, [virtualItems]);

  // Scroll to bottom on initial load and when channel changes
  const initialScrollDoneRef = useRef(false);
  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container || virtualItems.length === 0) return;

    // Skip if this is a scroll-preservation update (messages loaded via infinite scroll)
    if (scrollAnchorRef.current) return;

    if (!initialScrollDoneRef.current) {
      initialScrollDoneRef.current = true;

      // Skip scroll-to-bottom when a ?msg= deep link is present —
      // the ?msg= handler will scroll to the target message instead.
      if (highlightedMsgIdRef.current) {
        // Clear guard after 3 rAFs (same timing as normal initial scroll)
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            requestAnimationFrame(() => {
              programmaticScrollRef.current = false;
              console.debug(
                `[scroll:settled:deeplink] ${JSON.stringify({ scrollTop: container.scrollTop, scrollHeight: container.scrollHeight })}`,
              );
            });
          });
        });
        return;
      }

      console.debug(
        `[scroll:initialScroll] ${JSON.stringify({ count: virtualItems.length, scrollTop: container.scrollTop, scrollHeight: container.scrollHeight })}`,
      );
      virtualizer.scrollToOffset(virtualizer.getTotalSize());

      let prevHeight = -1;
      let attempts = 0;
      const settle = () => {
        requestAnimationFrame(() => {
          container.scrollTop = container.scrollHeight;
          if (container.scrollHeight !== prevHeight && attempts < 10) {
            prevHeight = container.scrollHeight;
            attempts++;
            settle();
          } else {
            programmaticScrollRef.current = false;
            console.debug(
              `[scroll:settled] ${JSON.stringify({ scrollTop: container.scrollTop, scrollHeight: container.scrollHeight, gap: container.scrollHeight - container.scrollTop - container.clientHeight, attempts })}`,
            );
          }
        });
      };
      settle();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally omit virtualizer; we only want this on virtualItems change
  }, [virtualItems]);

  // Reset state when channel changes
  useEffect(() => {
    initialScrollDoneRef.current = false;
    programmaticScrollRef.current = true; // Block observer for new channel's initial scroll
    scrollAnchorRef.current = null;
    lastBottomMessageIdRef.current = null;
    wasFetchingNextRef.current = false;
    wasFetchingPrevRef.current = false;
    setNewMessageCount(0);
    setShowJumpButton(false);
    isAtBottomRef.current = true;
  }, [channelId]);

  // Re-snap scroll to bottom when container resizes. The composer is a sibling in
  // a flex column, so typing indicators, attachment previews, or window resizes
  // change this container's height. Without this, the bottom message would be
  // pushed behind the composer until the next message arrives.
  // Dep: isLoading is needed because containerRef.current is null during the
  // loading early-return; the effect must re-run when the container mounts.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const ro = new ResizeObserver(() => {
      if (isAtBottomRef.current) {
        container.scrollTop = container.scrollHeight;
      }
    });
    ro.observe(container);
    return () => ro.disconnect();
  }, [channelId, isLoading]);

  // Auto-scroll to bottom when new messages arrive or content height changes
  // (e.g. link preview loads on an existing message) and user is at bottom.
  // Only when at the live edge (no newer pages to load) — otherwise loading
  // older pages via fetchPreviousPage would snap the scroll to the bottom.
  const totalSize = virtualizer.getTotalSize();
  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container || !initialScrollDoneRef.current) return;

    if (isAtBottomRef.current && !hasPreviousPageRef.current) {
      container.scrollTop = container.scrollHeight;
    }
  }, [allMessages.length, totalSize]);

  // Jump to message from ?msg= search param
  const jumpInProgressRef = useRef<string | null>(null);

  const scrollToAndHighlight = useCallback(
    (messageId: string, index?: number) => {
      const targetIndex =
        index ??
        virtualItems.findIndex((item) => item.type === 'message' && item.key === messageId);
      if (targetIndex === -1) return;

      virtualizer.scrollToIndex(targetIndex, { align: 'center' });

      // Highlight after scroll settles
      requestAnimationFrame(() => {
        setTimeout(() => {
          const el = document.getElementById(`message-${messageId}`);
          if (el) {
            el.classList.add('search-highlight');
            setTimeout(() => {
              el.classList.remove('search-highlight');
              // Clean up the ?msg param
              setSearchParams(
                (prev) => {
                  const next = new URLSearchParams(prev);
                  next.delete('msg');
                  return next;
                },
                { replace: true },
              );
              jumpInProgressRef.current = null;
            }, 2000);
          } else {
            jumpInProgressRef.current = null;
          }
        }, 100);
      });
    },
    [virtualItems, virtualizer, setSearchParams],
  );

  useEffect(() => {
    if (!highlightedMsgId || jumpInProgressRef.current === highlightedMsgId) return;
    jumpInProgressRef.current = highlightedMsgId;

    (async () => {
      const result = await jumpToMessage(highlightedMsgId);

      // If already loaded, we can scroll immediately
      if (result?.alreadyLoaded) {
        scrollToAndHighlight(highlightedMsgId);
      }
    })();
  }, [highlightedMsgId, jumpToMessage, scrollToAndHighlight]);

  // Once virtualItems update after a jump, scroll to the target
  useEffect(() => {
    if (!jumpInProgressRef.current) return;
    const targetId = jumpInProgressRef.current;
    const targetIndex = virtualItems.findIndex(
      (item) => item.type === 'message' && item.key === targetId,
    );
    if (targetIndex !== -1) {
      scrollToAndHighlight(targetId, targetIndex);
    }
  }, [virtualItems, scrollToAndHighlight]);

  // Handle jump to latest button click
  const handleJumpToLatest = useCallback(() => {
    console.debug(
      `[scroll:jumpToLatest] ${JSON.stringify({ isDetached, hasPreviousPage, path: isDetached || hasPreviousPage ? 'detached' : 'local' })}`,
    );
    if (isDetached || hasPreviousPage) {
      programmaticScrollRef.current = true; // Block observer — cleared by initial scroll rAF chain
      jumpToLatest();
      // Reset so initial scroll logic runs again for the fresh data
      initialScrollDoneRef.current = false;
    } else {
      programmaticScrollRef.current = true;
      const container = containerRef.current;
      virtualizer.scrollToOffset(virtualizer.getTotalSize());
      let prevHeight = -1;
      let attempts = 0;
      const settle = () => {
        requestAnimationFrame(() => {
          if (!container) return;
          container.scrollTop = container.scrollHeight;
          if (container.scrollHeight !== prevHeight && attempts < 10) {
            prevHeight = container.scrollHeight;
            attempts++;
            settle();
          } else {
            programmaticScrollRef.current = false;
            console.debug(
              `[scroll:settled:jumpLocal] ${JSON.stringify({ scrollTop: container?.scrollTop, scrollHeight: container?.scrollHeight, attempts })}`,
            );
          }
        });
      };
      settle();
    }
    const msgs = allMessagesRef.current;
    if (msgs.length > 0) {
      lastBottomMessageIdRef.current = msgs[msgs.length - 1].id;
    }
    setNewMessageCount(0);
    setShowJumpButton(false);
    isAtBottomRef.current = true;
    onAtBottomChange?.(true);
  }, [isDetached, hasPreviousPage, jumpToLatest, virtualizer, onAtBottomChange]);

  if (isLoading) {
    return (
      <div className="min-h-0 flex-1 overflow-hidden">
        <div className="flex h-full flex-col justify-end">
          {[...Array(8)].map((_, i) => (
            <MessageSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  if (allMessages.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center overflow-y-auto text-gray-500 dark:text-gray-400">
        <svg className="mb-4 h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
          />
        </svg>
        <p className="text-lg font-medium">No messages yet</p>
        <p className="text-sm">Be the first to send a message!</p>
      </div>
    );
  }

  const vRows = virtualizer.getVirtualItems();

  return (
    <div className="relative min-h-0 flex-1 overflow-hidden">
      <div ref={containerRef} onScroll={handleScroll} className="h-full overflow-y-auto">
        <div ref={topSentinelRef} className="h-px" />

        <div
          style={{
            height: totalSize,
            width: '100%',
            position: 'relative',
          }}
        >
          {vRows.map((vRow) => {
            const item = virtualItems[vRow.index];
            if (!item) return null;

            return (
              <div
                key={item.key}
                data-index={vRow.index}
                ref={virtualizer.measureElement}
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  transform: `translateY(${vRow.start}px)`,
                }}
              >
                <VirtualRow
                  item={item}
                  channelId={channelId}
                  channels={channelsData?.channels}
                  isAdmin={isAdmin}
                />
              </div>
            );
          })}
        </div>

        <div ref={bottomSentinelRef} className="h-px" />

        {/* Spacer below last message */}
        <div className="h-4" />
      </div>

      {/* Loading indicators — absolutely positioned to avoid layout shift */}
      {isFetchingNextPage && (
        <div className="pointer-events-none absolute top-0 right-0 left-0 z-10 flex justify-center py-2">
          <div className="rounded-full bg-white/90 px-3 py-1 text-xs text-gray-500 shadow-sm dark:bg-gray-800/90 dark:text-gray-400">
            Loading...
          </div>
        </div>
      )}
      {isFetchingPreviousPage && (
        <div className="pointer-events-none absolute right-0 bottom-0 left-0 z-10 flex justify-center py-2">
          <div className="rounded-full bg-white/90 px-3 py-1 text-xs text-gray-500 shadow-sm dark:bg-gray-800/90 dark:text-gray-400">
            Loading...
          </div>
        </div>
      )}

      {/* Jump to latest button */}
      {showJumpButton && (
        <JumpToLatestButton onClick={handleJumpToLatest} newMessageCount={newMessageCount} />
      )}
    </div>
  );
}

// Memoized row renderer
const VirtualRow = React.memo(function VirtualRow({
  item,
  channelId,
  channels,
  isAdmin,
}: {
  item: VirtualItem;
  channelId: string;
  channels?: ChannelWithMembership[];
  isAdmin?: boolean;
}) {
  switch (item.type) {
    case 'date-separator':
      return (
        <div className="flex items-center gap-4 px-4 py-3">
          <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
          <span className="text-xs font-medium text-gray-500 dark:text-gray-400">
            {formatDate(item.date)}
          </span>
          <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
        </div>
      );

    case 'unread-divider':
      return (
        <div className="flex items-center gap-4 px-4 py-3">
          <div className="h-px flex-1 bg-red-500" />
          <span className="text-xs font-medium text-red-500">New messages</span>
          <div className="h-px flex-1 bg-red-500" />
        </div>
      );

    case 'message':
      if (item.message.type === 'system') {
        return <SystemMessage message={item.message} channelId={channelId} />;
      }
      return (
        <MessageItem
          message={item.message}
          channelId={channelId}
          channels={channels}
          isAdmin={isAdmin}
        />
      );
  }
});
