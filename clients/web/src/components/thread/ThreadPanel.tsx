import { useState, useRef, type KeyboardEvent, type FormEvent } from 'react';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { useThreadMessages, useSendThreadReply, useAuth } from '../../hooks';
import { useUIStore } from '../../stores/uiStore';
import { Avatar, MessageSkeleton } from '../ui';
import { ReactionPicker } from '../message/ReactionPicker';
import { formatTime, cn } from '../../lib/utils';
import { messagesApi } from '../../api/messages';
import type { MessageWithUser, MessageListResult, Reaction } from '@feather/api-client';

function ClickableName({ userId, displayName }: { userId?: string; displayName: string }) {
  const { openProfile } = useUIStore();

  if (!userId) {
    return <span className="font-medium text-gray-900 dark:text-white">{displayName}</span>;
  }

  return (
    <button
      type="button"
      onClick={() => openProfile(userId)}
      className="font-medium text-gray-900 dark:text-white hover:underline cursor-pointer"
    >
      {displayName}
    </button>
  );
}

interface ThreadPanelProps {
  messageId: string;
}

export function ThreadPanel({ messageId }: ThreadPanelProps) {
  const queryClient = useQueryClient();
  const { closeThread } = useUIStore();
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useThreadMessages(messageId);

  // Get parent message from cache
  const parentMessage = getParentMessageFromCache(queryClient, messageId);

  // Flatten thread messages (already in chronological order from API)
  const threadMessages = data?.pages.flatMap((page) => page.messages) || [];

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
        <ParentMessage message={parentMessage} />
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
              <ThreadMessage key={message.id} message={message} parentMessageId={messageId} />
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

function ParentMessage({ message }: { message: MessageWithUser }) {
  const { openProfile } = useUIStore();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);
  const [localReactions, setLocalReactions] = useState(message.reactions || []);

  // Group reactions by emoji
  const reactionGroups = localReactions.reduce((acc, reaction) => {
    if (!acc[reaction.emoji]) {
      acc[reaction.emoji] = { emoji: reaction.emoji, count: 0, userIds: [], hasOwn: false };
    }
    acc[reaction.emoji].count++;
    acc[reaction.emoji].userIds.push(reaction.user_id);
    if (reaction.user_id === user?.id) {
      acc[reaction.emoji].hasOwn = true;
    }
    return acc;
  }, {} as Record<string, { emoji: string; count: number; userIds: string[]; hasOwn: boolean }>);

  const addReaction = useMutation({
    mutationFn: (emoji: string) => messagesApi.addReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id || 'temp';
      // Check if already reacted
      if (localReactions.some((r) => r.user_id === userId && r.emoji === emoji)) return;
      // Optimistic local update
      setLocalReactions((prev) => [
        ...prev,
        { id: 'temp', message_id: message.id, user_id: userId, emoji, created_at: new Date().toISOString() },
      ]);
      // Also update messages cache for consistency
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== message.id) return msg;
                const reactions = msg.reactions || [];
                if (reactions.some((r) => r.user_id === userId && r.emoji === emoji)) return msg;
                return {
                  ...msg,
                  reactions: [...reactions, { id: 'temp', message_id: message.id, user_id: userId, emoji, created_at: new Date().toISOString() }],
                };
              }),
            })),
          };
        }
      );
    },
  });

  const removeReaction = useMutation({
    mutationFn: (emoji: string) => messagesApi.removeReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id;
      // Optimistic local update
      setLocalReactions((prev) => prev.filter((r) => !(r.user_id === userId && r.emoji === emoji)));
      // Also update messages cache for consistency
      queryClient.setQueriesData(
        { queryKey: ['messages'] },
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== message.id) return msg;
                const reactions = msg.reactions || [];
                return {
                  ...msg,
                  reactions: reactions.filter((r) => !(r.user_id === userId && r.emoji === emoji)),
                };
              }),
            })),
          };
        }
      );
    },
  });

  const handleReactionClick = (emoji: string, hasOwn: boolean) => {
    if (hasOwn) {
      removeReaction.mutate(emoji);
    } else {
      addReaction.mutate(emoji);
    }
  };

  const handleAddReaction = (emoji: string) => {
    addReaction.mutate(emoji);
    setShowReactionPicker(false);
  };

  return (
    <div
      className="p-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 relative group"
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => {
        setShowActions(false);
        setShowReactionPicker(false);
      }}
    >
      <div className="flex items-start gap-3">
        <Avatar
          src={message.user_avatar_url}
          name={message.user_display_name || 'Unknown'}
          size="md"
          onClick={message.user_id ? () => openProfile(message.user_id!) : undefined}
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <ClickableName
              userId={message.user_id}
              displayName={message.user_display_name || 'Unknown User'}
            />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
          </div>
          <div className="text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
            {message.content}
          </div>

          {/* Reactions */}
          {Object.values(reactionGroups).length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {Object.values(reactionGroups).map(({ emoji, count, hasOwn }) => (
                <button
                  key={emoji}
                  onClick={() => handleReactionClick(emoji, hasOwn)}
                  className={cn(
                    'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-sm',
                    'border transition-colors',
                    hasOwn
                      ? 'bg-primary-100 dark:bg-primary-900/30 border-primary-300 dark:border-primary-700'
                      : 'bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600'
                  )}
                >
                  <span>{emoji}</span>
                  <span className="text-xs text-gray-600 dark:text-gray-300">{count}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
      <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
        {message.reply_count} {message.reply_count === 1 ? 'reply' : 'replies'}
      </div>

      {/* Action button */}
      {showActions && (
        <button
          onClick={() => setShowReactionPicker(!showReactionPicker)}
          className="absolute right-3 top-3 p-1 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded shadow-sm hover:bg-gray-100 dark:hover:bg-gray-600"
          title="Add reaction"
        >
          <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </button>
      )}

      {/* Reaction picker */}
      {showReactionPicker && (
        <div className="absolute right-3 top-12 z-10">
          <ReactionPicker onSelect={handleAddReaction} />
        </div>
      )}
    </div>
  );
}

interface ThreadMessageProps {
  message: MessageWithUser;
  parentMessageId: string;
}

function ThreadMessage({ message, parentMessageId }: ThreadMessageProps) {
  const { openProfile } = useUIStore();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);

  // Group reactions by emoji
  const reactionGroups = (message.reactions || []).reduce((acc, reaction) => {
    if (!acc[reaction.emoji]) {
      acc[reaction.emoji] = { emoji: reaction.emoji, count: 0, userIds: [], hasOwn: false };
    }
    acc[reaction.emoji].count++;
    acc[reaction.emoji].userIds.push(reaction.user_id);
    if (reaction.user_id === user?.id) {
      acc[reaction.emoji].hasOwn = true;
    }
    return acc;
  }, {} as Record<string, { emoji: string; count: number; userIds: string[]; hasOwn: boolean }>);

  const addReaction = useMutation({
    mutationFn: (emoji: string) => messagesApi.addReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id || 'temp';
      // Update thread cache
      queryClient.setQueryData(
        ['thread', parentMessageId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== message.id) return msg;
                const reactions = msg.reactions || [];
                if (reactions.some((r) => r.user_id === userId && r.emoji === emoji)) return msg;
                return {
                  ...msg,
                  reactions: [...reactions, { id: 'temp', message_id: message.id, user_id: userId, emoji, created_at: new Date().toISOString() }],
                };
              }),
            })),
          };
        }
      );
    },
  });

  const removeReaction = useMutation({
    mutationFn: (emoji: string) => messagesApi.removeReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id;
      queryClient.setQueryData(
        ['thread', parentMessageId],
        (old: { pages: MessageListResult[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== message.id) return msg;
                const reactions = msg.reactions || [];
                return {
                  ...msg,
                  reactions: reactions.filter((r) => !(r.user_id === userId && r.emoji === emoji)),
                };
              }),
            })),
          };
        }
      );
    },
  });

  const handleReactionClick = (emoji: string, hasOwn: boolean) => {
    if (hasOwn) {
      removeReaction.mutate(emoji);
    } else {
      addReaction.mutate(emoji);
    }
  };

  const handleAddReaction = (emoji: string) => {
    addReaction.mutate(emoji);
    setShowReactionPicker(false);
  };

  return (
    <div
      className="px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/50 relative group"
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => {
        setShowActions(false);
        setShowReactionPicker(false);
      }}
    >
      <div className="flex items-start gap-3">
        <Avatar
          src={message.user_avatar_url}
          name={message.user_display_name || 'Unknown'}
          size="sm"
          onClick={message.user_id ? () => openProfile(message.user_id!) : undefined}
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <ClickableName
              userId={message.user_id}
              displayName={message.user_display_name || 'Unknown User'}
            />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
          </div>
          <div className="text-sm text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
            {message.content}
          </div>

          {/* Reactions */}
          {Object.values(reactionGroups).length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {Object.values(reactionGroups).map(({ emoji, count, hasOwn }) => (
                <button
                  key={emoji}
                  onClick={() => handleReactionClick(emoji, hasOwn)}
                  className={cn(
                    'inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-xs',
                    'border transition-colors',
                    hasOwn
                      ? 'bg-primary-100 dark:bg-primary-900/30 border-primary-300 dark:border-primary-700'
                      : 'bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600'
                  )}
                >
                  <span>{emoji}</span>
                  <span className="text-gray-600 dark:text-gray-300">{count}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Action button */}
      {showActions && (
        <button
          onClick={() => setShowReactionPicker(!showReactionPicker)}
          className="absolute right-2 top-1 p-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded shadow-sm hover:bg-gray-100 dark:hover:bg-gray-700"
          title="Add reaction"
        >
          <svg className="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </button>
      )}

      {/* Reaction picker */}
      {showReactionPicker && (
        <div className="absolute right-2 top-8 z-10">
          <ReactionPicker onSelect={handleAddReaction} />
        </div>
      )}
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
