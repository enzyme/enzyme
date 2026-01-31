import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Avatar } from '../ui';
import { ReactionPicker } from './ReactionPicker';
import { useAuth, useAddReaction, useRemoveReaction } from '../../hooks';
import { useMarkMessageUnread } from '../../hooks/useMessages';
import { useUIStore } from '../../stores/uiStore';
import { formatTime, formatRelativeTime, cn } from '../../lib/utils';
import type { MessageWithUser } from '@feather/api-client';

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

interface MessageItemProps {
  message: MessageWithUser;
  channelId: string;
}

export function MessageItem({ message, channelId }: MessageItemProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);
  const { user } = useAuth();
  const { openThread, openProfile } = useUIStore();
  const addReaction = useAddReaction(channelId);
  const removeReaction = useRemoveReaction(channelId);
  const markUnread = useMarkMessageUnread(workspaceId || '');

  const isDeleted = !!message.deleted_at;
  const isEdited = !!message.edited_at;
  const isOwnMessage = user?.id === message.user_id;

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

  const handleReactionClick = (emoji: string, hasOwn: boolean) => {
    if (hasOwn) {
      removeReaction.mutate({ messageId: message.id, emoji });
    } else {
      addReaction.mutate({ messageId: message.id, emoji });
    }
  };

  const handleAddReaction = (emoji: string) => {
    addReaction.mutate({ messageId: message.id, emoji });
    setShowReactionPicker(false);
  };

  if (isDeleted) {
    return (
      <div className="px-4 py-2 text-gray-400 dark:text-gray-500 italic text-sm">
        This message was deleted
      </div>
    );
  }

  return (
    <div
      className={cn(
        'group px-4 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/50',
        'relative'
      )}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => {
        setShowActions(false);
        setShowReactionPicker(false);
      }}
    >
      <div className="flex items-start gap-3">
        {/* Avatar */}
        <Avatar
          src={message.user_avatar_url}
          name={message.user_display_name || 'Unknown'}
          size="md"
          onClick={message.user_id ? () => openProfile(message.user_id!) : undefined}
        />

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex items-baseline gap-2">
            <ClickableName
              userId={message.user_id}
              displayName={message.user_display_name || 'Unknown User'}
            />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
            {isEdited && (
              <span className="text-xs text-gray-400 dark:text-gray-500">(edited)</span>
            )}
          </div>

          {/* Message content */}
          <div className="text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
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

          {/* Thread replies indicator */}
          {message.reply_count > 0 && (
            <button
              onClick={() => openThread(message.id)}
              className="mt-2 flex items-center gap-2 group/thread hover:bg-gray-100 dark:hover:bg-gray-700/50 rounded px-1 -mx-1 py-0.5"
            >
              {/* Avatar stack */}
              {message.thread_participants && message.thread_participants.length > 0 && (
                <div className="flex -space-x-1.5">
                  {message.thread_participants.map((participant) => (
                    <Avatar
                      key={participant.user_id}
                      src={participant.avatar_url}
                      name={participant.display_name || 'User'}
                      size="xs"
                      className="ring-2 ring-white dark:ring-gray-900"
                    />
                  ))}
                </div>
              )}
              <span className="text-sm text-primary-600 dark:text-primary-400 group-hover/thread:underline">
                {message.reply_count} {message.reply_count === 1 ? 'reply' : 'replies'}
              </span>
              {message.last_reply_at && (
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  Last reply {formatRelativeTime(message.last_reply_at)}
                </span>
              )}
            </button>
          )}
        </div>
      </div>

      {/* Action buttons */}
      {showActions && (
        <div className="absolute right-4 top-0 -translate-y-1/2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm flex items-center">
          <button
            onClick={() => setShowReactionPicker(!showReactionPicker)}
            className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-l-lg"
            title="Add reaction"
          >
            <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </button>

          <button
            onClick={() => openThread(message.id)}
            className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
            title="Reply in thread"
          >
            <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
            </svg>
          </button>

          <button
            onClick={() => markUnread.mutate(message.id)}
            className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
            title="Mark unread"
          >
            <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          </button>

          {isOwnMessage && (
            <button
              className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-r-lg"
              title="More actions"
            >
              <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
              </svg>
            </button>
          )}
        </div>
      )}

      {/* Reaction picker */}
      {showReactionPicker && (
        <div className="absolute right-4 top-8 z-10">
          <ReactionPicker onSelect={handleAddReaction} />
        </div>
      )}
    </div>
  );
}
