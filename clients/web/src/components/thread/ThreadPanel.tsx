import { useState, useRef, type KeyboardEvent, type FormEvent } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useThreadMessages, useSendThreadReply } from '../../hooks';
import { useUIStore } from '../../stores/uiStore';
import { Avatar, MessageSkeleton } from '../ui';
import { formatTime, cn } from '../../lib/utils';
import type { MessageWithUser, MessageListResult } from '@feather/api-client';

interface ThreadPanelProps {
  messageId: string;
}

export function ThreadPanel({ messageId }: ThreadPanelProps) {
  const queryClient = useQueryClient();
  const { closeThread } = useUIStore();
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useThreadMessages(messageId);

  // Get parent message from cache
  const parentMessage = getParentMessageFromCache(queryClient, messageId);

  // Flatten and reverse thread messages
  const threadMessages = data?.pages.flatMap((page) => page.messages).reverse() || [];

  return (
    <div className="w-96 border-l border-gray-200 dark:border-gray-700 flex flex-col bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
        <h3 className="font-semibold text-gray-900 dark:text-white">Thread</h3>
        <button
          onClick={closeThread}
          className="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Parent message */}
      {parentMessage && (
        <div className="p-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <div className="flex items-start gap-3">
            <Avatar
              src={parentMessage.user_avatar_url}
              name={parentMessage.user_display_name || 'Unknown'}
              size="md"
            />
            <div className="flex-1 min-w-0">
              <div className="flex items-baseline gap-2">
                <span className="font-medium text-gray-900 dark:text-white">
                  {parentMessage.user_display_name || 'Unknown User'}
                </span>
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {formatTime(parentMessage.created_at)}
                </span>
              </div>
              <div className="text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
                {parentMessage.content}
              </div>
            </div>
          </div>
          <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
            {parentMessage.reply_count} {parentMessage.reply_count === 1 ? 'reply' : 'replies'}
          </div>
        </div>
      )}

      {/* Thread messages */}
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <>
            <MessageSkeleton />
            <MessageSkeleton />
          </>
        ) : (
          <>
            {hasNextPage && (
              <button
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
                className="w-full py-2 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400"
              >
                {isFetchingNextPage ? 'Loading...' : 'Load more replies'}
              </button>
            )}

            {threadMessages.map((message) => (
              <ThreadMessage key={message.id} message={message} />
            ))}
          </>
        )}
      </div>

      {/* Reply composer */}
      {parentMessage && (
        <ThreadComposer
          parentMessageId={messageId}
          channelId={parentMessage.channel_id}
        />
      )}
    </div>
  );
}

function ThreadMessage({ message }: { message: MessageWithUser }) {
  return (
    <div className="px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/50">
      <div className="flex items-start gap-3">
        <Avatar
          src={message.user_avatar_url}
          name={message.user_display_name || 'Unknown'}
          size="sm"
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <span className="font-medium text-sm text-gray-900 dark:text-white">
              {message.user_display_name || 'Unknown User'}
            </span>
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
          </div>
          <div className="text-sm text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
            {message.content}
          </div>
        </div>
      </div>
    </div>
  );
}

interface ThreadComposerProps {
  parentMessageId: string;
  channelId: string;
}

function ThreadComposer({ parentMessageId, channelId }: ThreadComposerProps) {
  const [content, setContent] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const sendReply = useSendThreadReply(parentMessageId, channelId);

  const handleSubmit = async (e?: FormEvent) => {
    e?.preventDefault();

    const trimmedContent = content.trim();
    if (!trimmedContent || sendReply.isPending) return;

    try {
      await sendReply.mutateAsync(trimmedContent);
      setContent('');
      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto';
      }
    } catch {
      // Error handled by mutation
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleChange = (value: string) => {
    setContent(value);
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
    }
  };

  return (
    <div className="p-4 border-t border-gray-200 dark:border-gray-700">
      <form onSubmit={handleSubmit}>
        <textarea
          ref={textareaRef}
          value={content}
          onChange={(e) => handleChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Reply..."
          rows={1}
          className={cn(
            'w-full px-3 py-2 resize-none',
            'border border-gray-300 dark:border-gray-600 rounded-lg',
            'bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm',
            'placeholder-gray-500 dark:placeholder-gray-400',
            'focus:outline-none focus:ring-2 focus:ring-primary-500',
            'max-h-24'
          )}
        />
      </form>
    </div>
  );
}

function getParentMessageFromCache(
  queryClient: ReturnType<typeof useQueryClient>,
  messageId: string
): MessageWithUser | undefined {
  // Search through all message queries
  const queries = queryClient.getQueriesData<{
    pages: MessageListResult[];
    pageParams: (string | undefined)[];
  }>({ queryKey: ['messages'] });

  for (const [, data] of queries) {
    if (!data) continue;
    for (const page of data.pages) {
      const found = page.messages.find((m) => m.id === messageId);
      if (found) return found;
    }
  }

  return undefined;
}
