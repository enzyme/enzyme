import { useState, useRef, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { Button as AriaButton } from 'react-aria-components';
import {
  FaceSmileIcon,
  ChatBubbleBottomCenterTextIcon,
  EyeSlashIcon,
  EllipsisVerticalIcon,
  PencilSquareIcon,
  TrashIcon,
  LinkIcon,
} from '@heroicons/react/24/outline';
import { Avatar, Menu, MenuItem, Modal, Button, Tooltip, toast } from '../ui';
import { ReactionPicker } from './ReactionPicker';
import { AttachmentDisplay } from './AttachmentDisplay';
import { MessageContent } from './MessageContent';
import { ThreadRepliesIndicator } from './ThreadRepliesIndicator';
import { useAuth, useAddReaction, useRemoveReaction, useWorkspaceMembers } from '../../hooks';
import { useMarkMessageUnread, useUpdateMessage, useDeleteMessage } from '../../hooks/useMessages';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { cn, formatTime } from '../../lib/utils';
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
  const [isDeleting, setIsDeleting] = useState(false);
  const editTextareaRef = useRef<HTMLTextAreaElement>(null);
  const messageRef = useRef<HTMLDivElement>(null);
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

  const handleCopyLink = () => {
    const url = `${window.location.origin}/w/${workspaceId}/c/${channelId}?msg=${message.id}`;
    navigator.clipboard.writeText(url);
    toast('Link copied to clipboard', 'success');
    setShowDropdown(false);
  };

  const handleDeleteConfirm = () => {
    setShowDeleteModal(false);
    // Messages with replies: delete immediately (shows placeholder)
    // Messages without replies: animate then delete
    if (message.reply_count > 0) {
      deleteMessage.mutate(message.id);
    } else {
      // Capture current height before animating
      if (messageRef.current) {
        const height = messageRef.current.offsetHeight;
        messageRef.current.style.maxHeight = `${height}px`;
        // Force reflow to ensure the maxHeight is applied before transition
        void messageRef.current.offsetHeight;
      }
      setIsDeleting(true);
      // Delay the actual deletion to allow animation to play
      setTimeout(() => {
        deleteMessage.mutate(message.id);
      }, 500);
    }
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

  // Deleted messages without replies: render nothing
  if (isDeleted && message.reply_count === 0) {
    return null;
  }

  // Deleted messages with replies: show placeholder with thread indicator
  if (isDeleted) {
    return (
      <div className="group px-4 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/50">
        <div className="flex items-start gap-3">
          {/* Trash icon in avatar-sized circle */}
          <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
            <TrashIcon className="w-4 h-4 text-gray-400 dark:text-gray-500" />
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <span className="text-sm text-gray-400 dark:text-gray-500 italic">
              This message was deleted.
            </span>

            {/* Thread replies indicator */}
            <ThreadRepliesIndicator
              messageId={message.id}
              replyCount={message.reply_count}
              lastReplyAt={message.last_reply_at}
              threadParticipants={message.thread_participants}
            />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={messageRef}
      className={cn(
        "group px-4 py-1.5 relative",
        isDeleting
          ? "bg-red-400 dark:bg-red-700 !max-h-0 opacity-0 !py-0 overflow-hidden transition-all duration-500"
          : "hover:bg-gray-50 dark:hover:bg-gray-800/50",
        showDropdown && !isDeleting && "bg-gray-50 dark:bg-gray-800/50"
      )}
      style={isDeleting ? { marginTop: 0, marginBottom: 0 } : undefined}
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
          id={message.user_id}
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
          <ThreadRepliesIndicator
            messageId={message.id}
            replyCount={message.reply_count}
            lastReplyAt={message.last_reply_at}
            threadParticipants={message.thread_participants}
          />
        </div>
      </div>

      {/* Action buttons */}
      {showActions && !isEditing && (
        <div className="absolute right-4 top-0 -translate-y-1/2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm flex items-center">
          <Tooltip content="Add reaction">
            <AriaButton
              onPress={() => setShowReactionPicker(!showReactionPicker)}
              className="group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-l-lg"
            >
              <FaceSmileIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
            </AriaButton>
          </Tooltip>

          <Tooltip content="Reply in thread">
            <AriaButton
              onPress={() => openThread(message.id)}
              className="group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <ChatBubbleBottomCenterTextIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
            </AriaButton>
          </Tooltip>

          <Tooltip content="More options">
            <Menu
              open={showDropdown}
              onOpenChange={setShowDropdown}
              align="end"
              trigger={
                <AriaButton
                  className={cn(
                    'group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-r-lg',
                    showDropdown && 'bg-gray-100 dark:bg-gray-700'
                  )}
                  aria-label="More options"
                >
                  <EllipsisVerticalIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
                </AriaButton>
              }
            >
              <MenuItem
                onAction={handleCopyLink}
                icon={<LinkIcon className="w-4 h-4" />}
              >
                Copy link to message
              </MenuItem>
              <MenuItem
                onAction={() => markUnread.mutate(message.id)}
                icon={<EyeSlashIcon className="w-4 h-4" />}
              >
                Mark unread
              </MenuItem>
              {isOwnMessage && (
                <>
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
                </>
              )}
            </Menu>
          </Tooltip>
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
