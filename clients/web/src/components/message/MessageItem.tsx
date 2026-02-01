import { useState, useRef, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { Button as AriaButton } from 'react-aria-components';
import {
  FaceSmileIcon,
  ArrowUturnLeftIcon,
  EnvelopeIcon,
  EllipsisVerticalIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { Avatar, AvatarStack, Menu, MenuItem, Modal, Button, Tooltip } from '../ui';
import { ReactionPicker } from './ReactionPicker';
import { AttachmentDisplay } from './AttachmentDisplay';
import { MessageContent } from './MessageContent';
import { useAuth, useAddReaction, useRemoveReaction, useWorkspaceMembers } from '../../hooks';
import { useMarkMessageUnread, useUpdateMessage, useDeleteMessage } from '../../hooks/useMessages';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { cn, formatTime, formatRelativeTime } from '../../lib/utils';
import type { MessageWithUser } from '@feather/api-client';

function ClickableName({ userId, displayName }: { userId?: string; displayName: string }) {
  const { openProfile } = useProfilePanel();

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
  const [showDropdown, setShowDropdown] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState('');
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const editTextareaRef = useRef<HTMLTextAreaElement>(null);
  const { user } = useAuth();
  const { openThread } = useThreadPanel();
  const { openProfile } = useProfilePanel();
  const addReaction = useAddReaction(channelId);
  const removeReaction = useRemoveReaction(channelId);
  const markUnread = useMarkMessageUnread(workspaceId || '');
  const updateMessage = useUpdateMessage();
  const deleteMessage = useDeleteMessage();
  const { data: membersData } = useWorkspaceMembers(workspaceId);

  // Create a lookup map from user ID to display name
  const memberNames = (membersData?.members || []).reduce((acc, member) => {
    acc[member.user_id] = member.display_name;
    return acc;
  }, {} as Record<string, string>);

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

  const handleStartEdit = () => {
    setEditContent(message.content);
    setIsEditing(true);
    setShowDropdown(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditContent('');
  };

  const handleSaveEdit = () => {
    if (editContent.trim() && editContent.trim() !== message.content) {
      updateMessage.mutate({ messageId: message.id, content: editContent.trim() });
    }
    setIsEditing(false);
    setEditContent('');
  };

  const handleDeleteClick = () => {
    setShowDropdown(false);
    setShowDeleteModal(true);
  };

  const handleDeleteConfirm = () => {
    deleteMessage.mutate(message.id);
    setShowDeleteModal(false);
  };

  const handleEditKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Escape') {
      handleCancelEdit();
    } else if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSaveEdit();
    }
  };

  // Auto-focus textarea when entering edit mode
  useEffect(() => {
    if (isEditing && editTextareaRef.current) {
      editTextareaRef.current.focus();
      // Move cursor to end
      editTextareaRef.current.selectionStart = editTextareaRef.current.value.length;
    }
  }, [isEditing]);

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
        "group px-4 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/50 relative",
        showDropdown && "bg-gray-50 dark:bg-gray-800/50"
      )}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => {
        if (!showDropdown) {
          setShowActions(false);
        }
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
          {isEditing ? (
            <div className="space-y-2 mt-1">
              <textarea
                ref={editTextareaRef}
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                onKeyDown={handleEditKeyDown}
                className="w-full p-2 border border-gray-300 dark:border-gray-600 rounded-lg resize-none bg-white dark:bg-gray-700 text-gray-800 dark:text-gray-200 focus:outline-none focus:ring-2 focus:ring-primary-500"
                rows={3}
              />
              <div className="flex items-center gap-2 text-sm">
                <button
                  onClick={handleSaveEdit}
                  disabled={updateMessage.isPending || !editContent.trim()}
                  className="px-3 py-1 bg-primary-600 text-white rounded hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {updateMessage.isPending ? 'Saving...' : 'Save'}
                </button>
                <button
                  onClick={handleCancelEdit}
                  className="px-3 py-1 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                >
                  Cancel
                </button>
                <span className="text-gray-500 dark:text-gray-400 text-xs">
                  Escape to cancel, Enter to save
                </span>
              </div>
            </div>
          ) : (
            <>
              {message.content && (
                <div className="text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
                  <MessageContent
                    content={message.content}
                    members={membersData?.members}
                  />
                </div>
              )}
              {message.attachments && message.attachments.length > 0 && (
                <AttachmentDisplay attachments={message.attachments} />
              )}
            </>
          )}

          {/* Reactions */}
          {Object.values(reactionGroups).length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {Object.values(reactionGroups).map(({ emoji, count, userIds, hasOwn }) => {
                const userNames = userIds
                  .map((id) => memberNames[id] || 'Unknown')
                  .join(', ');
                return (
                  <Tooltip key={emoji} content={userNames}>
                    <AriaButton
                      onPress={() => handleReactionClick(emoji, hasOwn)}
                      className={cn(
                        'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-sm border transition-colors',
                        hasOwn
                          ? 'bg-primary-100 dark:bg-primary-900/30 border-primary-300 dark:border-primary-700'
                          : 'bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600'
                      )}
                    >
                      <span>{emoji}</span>
                      <span className="text-xs text-gray-600 dark:text-gray-300">{count}</span>
                    </AriaButton>
                  </Tooltip>
                );
              })}
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
                <AvatarStack users={message.thread_participants} showCount={false} />
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
      {showActions && !isEditing && (
        <div className="absolute right-4 top-0 -translate-y-1/2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm flex items-center">
          <button
            onClick={() => setShowReactionPicker(!showReactionPicker)}
            className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-l-lg"
            title="Add reaction"
          >
            <FaceSmileIcon className="w-4 h-4 text-gray-500" />
          </button>

          <button
            onClick={() => openThread(message.id)}
            className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
            title="Reply in thread"
          >
            <ArrowUturnLeftIcon className="w-4 h-4 text-gray-500" />
          </button>

          <button
            onClick={() => markUnread.mutate(message.id)}
            className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
            title="Mark unread"
          >
            <EnvelopeIcon className="w-4 h-4 text-gray-500" />
          </button>

          {isOwnMessage && (
            <Menu
              open={showDropdown}
              onOpenChange={setShowDropdown}
              align="end"
              trigger={
                <AriaButton
                  className={cn(
                    'p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-r-lg',
                    showDropdown && 'bg-gray-100 dark:bg-gray-700'
                  )}
                  aria-label="More actions"
                >
                  <EllipsisVerticalIcon className="w-4 h-4 text-gray-500" />
                </AriaButton>
              }
            >
              <MenuItem
                onAction={handleStartEdit}
                icon={<PencilSquareIcon className="w-4 h-4" />}
              >
                Edit message
              </MenuItem>
              <MenuItem
                onAction={handleDeleteClick}
                variant="danger"
                icon={<TrashIcon className="w-4 h-4" />}
              >
                Delete message
              </MenuItem>
            </Menu>
          )}
        </div>
      )}

      {/* Reaction picker */}
      {showReactionPicker && (
        <div className="absolute right-4 top-8 z-10">
          <ReactionPicker onSelect={handleAddReaction} />
        </div>
      )}

      {/* Delete confirmation modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="Delete message"
        size="sm"
      >
        <p className="text-gray-600 dark:text-gray-300 mb-4">
          Are you sure you want to delete this message? This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <Button
            variant="secondary"
            onPress={() => setShowDeleteModal(false)}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            onPress={handleDeleteConfirm}
            isLoading={deleteMessage.isPending}
          >
            Delete
          </Button>
        </div>
      </Modal>
    </div>
  );
}
