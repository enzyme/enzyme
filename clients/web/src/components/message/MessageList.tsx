import { useEffect, useRef, useCallback } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { useMessages, useChannels } from '../../hooks';
import { usePrewarmSignedUrls } from '../../hooks/usePrewarmSignedUrls';
import { MessageItem } from './MessageItem';
import { SystemMessage } from './SystemMessage';
import { MessageSkeleton } from '../ui';
import { formatDate } from '../../lib/utils';
import type { MessageWithUser } from '@feather/api-client';

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
  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage } =
    useMessages(channelId);
  const { data: channelsData } = useChannels(workspaceId);

  // Pre-warm signed URL cache for any file attachments in loaded messages
  usePrewarmSignedUrls(data?.pages);

  const containerRef = useRef<HTMLDivElement>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const prevScrollHeightRef = useRef(0);
  const isAtBottomRef = useRef(true);

  // Flatten messages from all pages (they come newest-first)
  const allMessages = data?.pages.flatMap((page) => page.messages) || [];

  // Reverse for display (oldest at top)
  const messages = [...allMessages].reverse();

  // Find the index of the last read message in the flat list
  const lastReadIndex = lastReadMessageId
    ? messages.findIndex((m) => m.id === lastReadMessageId)
    : -1;

  // Group messages by date
  const messagesByDate = messages.reduce(
    (acc, msg) => {
      const dateKey = new Date(msg.created_at).toDateString();
      if (!acc[dateKey]) {
        acc[dateKey] = [];
      }
      acc[dateKey].push(msg);
      return acc;
    },
    {} as Record<string, MessageWithUser[]>,
  );

  // Check if user is at bottom
  const checkIfAtBottom = useCallback(() => {
    const container = containerRef.current;
    if (!container) return true;
    const { scrollTop, scrollHeight, clientHeight } = container;
    return scrollHeight - scrollTop - clientHeight < 100;
  }, []);

  // Handle scroll to detect when to load more
  const handleScroll = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;

    const atBottom = checkIfAtBottom();
    isAtBottomRef.current = atBottom;
    onAtBottomChange?.(atBottom);

    // Load more when scrolled near top
    if (container.scrollTop < 100 && hasNextPage && !isFetchingNextPage) {
      prevScrollHeightRef.current = container.scrollHeight;
      fetchNextPage();
    }
  }, [checkIfAtBottom, hasNextPage, isFetchingNextPage, fetchNextPage, onAtBottomChange]);

  // Preserve scroll position when prepending older messages
  useEffect(() => {
    const container = containerRef.current;
    if (!container || prevScrollHeightRef.current === 0) return;

    const newScrollHeight = container.scrollHeight;
    const scrollDiff = newScrollHeight - prevScrollHeightRef.current;
    container.scrollTop += scrollDiff;
    prevScrollHeightRef.current = 0;
  }, [data?.pages.length]);

  // Scroll to bottom on initial load and new messages (if at bottom)
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    if (isAtBottomRef.current || messages.length <= 50) {
      container.scrollTop = container.scrollHeight;
    }
  }, [messages.length]);

  // Jump to message from ?msg= search param
  const [searchParams, setSearchParams] = useSearchParams();
  const highlightedMsgId = searchParams.get('msg');

  useEffect(() => {
    if (!highlightedMsgId || messages.length === 0) return;

    const el = document.getElementById(`message-${highlightedMsgId}`);
    if (el) {
      el.scrollIntoView({ block: 'center', behavior: 'smooth' });
      el.classList.add('search-highlight');
      const timer = setTimeout(() => {
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
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [highlightedMsgId, messages.length, setSearchParams]);

  // Intersection observer for infinite scroll
  useEffect(() => {
    const loadMoreElement = loadMoreRef.current;
    if (!loadMoreElement) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          const container = containerRef.current;
          if (container) {
            prevScrollHeightRef.current = container.scrollHeight;
          }
          fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    observer.observe(loadMoreElement);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (isLoading) {
    return (
      <div className="flex-1 overflow-y-auto">
        {[...Array(5)].map((_, i) => (
          <MessageSkeleton key={i} />
        ))}
      </div>
    );
  }

  return (
    <div ref={containerRef} onScroll={handleScroll} className="flex-1 overflow-y-auto">
      {/* Load more trigger */}
      <div ref={loadMoreRef} className="h-1" />

      {isFetchingNextPage && (
        <div className="py-4 text-center">
          <MessageSkeleton />
        </div>
      )}

      {messages.length === 0 ? (
        <div className="flex h-full flex-col items-center justify-center text-gray-500 dark:text-gray-400">
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
      ) : (
        Object.entries(messagesByDate).map(([date, msgs]) => (
          <div key={date}>
            {/* Date separator */}
            <div className="flex items-center gap-4 px-4 py-3">
              <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
              <span className="text-xs font-medium text-gray-500 dark:text-gray-400">
                {formatDate(msgs[0].created_at)}
              </span>
              <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
            </div>

            {/* Messages */}
            {msgs.map((message) => {
              // Check if we should show the unread divider after this message
              // Show divider after the last read message if there are unread messages after it
              const messageIndex = messages.findIndex((m) => m.id === message.id);
              const showUnreadDivider =
                unreadCount > 0 &&
                lastReadIndex !== -1 &&
                messageIndex === lastReadIndex &&
                messageIndex < messages.length - 1; // There are messages after this one

              // Check if this is a system message
              const isSystemMessage = message.type === 'system';

              return (
                <div key={message.id}>
                  {isSystemMessage ? (
                    <SystemMessage message={message} channelId={channelId} />
                  ) : (
                    <MessageItem
                      message={message}
                      channelId={channelId}
                      channels={channelsData?.channels}
                    />
                  )}
                  {showUnreadDivider && (
                    <div className="flex items-center gap-4 px-4 py-3">
                      <div className="h-px flex-1 bg-red-500" />
                      <span className="text-xs font-medium text-red-500">New messages</span>
                      <div className="h-px flex-1 bg-red-500" />
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        ))
      )}

      {/* Spacer below last message */}
      {messages.length > 0 && <div className="h-4" />}
    </div>
  );
}
