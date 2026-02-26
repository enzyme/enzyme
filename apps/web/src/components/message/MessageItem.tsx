import { useState, useRef, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  TrashIcon,
  ChatBubbleBottomCenterTextIcon,
  LinkIcon,
  EyeSlashIcon,
  PencilSquareIcon,
  MapPinIcon,
  NoSymbolIcon,
} from '@heroicons/react/24/outline';
import { UserIcon, ChatBubbleLeftIcon } from '@heroicons/react/24/outline';
import {
  Avatar,
  Modal,
  Button,
  toast,
  ContextMenu,
  useContextMenu,
  MenuItem,
  MenuSeparator,
} from '../ui';
import { LazyRichTextEditor, useEditorMembers, useEditorChannels } from '../editor';
import type { RichTextEditorRef } from '../editor';
import { AttachmentDisplay } from './AttachmentDisplay';
import { LinkPreviewDisplay } from './LinkPreviewDisplay';
import { MessagePreviewDisplay } from './MessagePreviewDisplay';
import { CollapsibleMessage } from './CollapsibleMessage';
import { MessageContent } from './MessageContent';
import { ThreadRepliesIndicator } from './ThreadRepliesIndicator';
import { MessageActionBar } from './MessageActionBar';
import { ReactionsDisplay } from './ReactionsDisplay';
import { groupReactionsByEmoji, createMemberNamesMap } from './reactionUtils';
import { useAuth, useAddReaction, useRemoveReaction, useWorkspaceMembers } from '../../hooks';
import { useCreateDM } from '../../hooks/useChannels';
import {
  useMarkMessageUnread,
  useUpdateMessage,
  useDeleteMessage,
  useDeleteLinkPreview,
} from '../../hooks/useMessages';
import { usePinMessage, useUnpinMessage, useBlockUser } from '../../hooks/useModeration';
import { useCustomEmojiMap, useCustomEmojis } from '../../hooks/useCustomEmojis';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { cn, formatTime } from '../../lib/utils';
import type { MessageWithUser, ChannelWithMembership } from '@enzyme/api-client';

function ClickableName({
  userId,
  displayName,
  onContextMenu,
}: {
  userId?: string;
  displayName: string;
  onContextMenu?: (e: React.MouseEvent) => void;
}) {
  const { openProfile } = useProfilePanel();

  if (!userId) {
    return <span className="font-medium text-gray-900 dark:text-white">{displayName}</span>;
  }

  return (
    <button
      type="button"
      onClick={() => openProfile(userId)}
      onContextMenu={onContextMenu}
      className="cursor-pointer font-medium text-gray-900 hover:underline dark:text-white"
    >
      {displayName}
    </button>
  );
}

interface MessageItemProps {
  message: MessageWithUser;
  channelId: string;
  channels?: ChannelWithMembership[];
  isAdmin?: boolean;
}

export function MessageItem({ message, channelId, channels, isAdmin }: MessageItemProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const customEmojiMap = useCustomEmojiMap(workspaceId);
  const { data: customEmojis } = useCustomEmojis(workspaceId);
  const [showActions, setShowActions] = useState(false);
  const [reactionPickerOpen, setReactionPickerOpen] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const editEditorRef = useRef<RichTextEditorRef>(null);
  const messageRef = useRef<HTMLDivElement>(null);
  const { user } = useAuth();
  const { openThread } = useThreadPanel();
  const { openProfile } = useProfilePanel();
  const addReaction = useAddReaction(channelId);
  const removeReaction = useRemoveReaction(channelId);
  const markUnread = useMarkMessageUnread(workspaceId || '');
  const updateMessage = useUpdateMessage();
  const deleteMessage = useDeleteMessage();
  const deleteLinkPreview = useDeleteLinkPreview();
  const pinMessage = usePinMessage(channelId);
  const unpinMessage = useUnpinMessage(channelId);
  const blockUser = useBlockUser(workspaceId);
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const navigate = useNavigate();
  const createDM = useCreateDM(workspaceId || '');
  const msgCtx = useContextMenu();
  const userCtx = useContextMenu();

  // Stop propagation so right-clicking avatar/name doesn't also open the message context menu
  const onUserContextMenu = (e: React.MouseEvent) => {
    e.stopPropagation();
    userCtx.onContextMenu(e);
  };

  const memberNames = createMemberNamesMap(membersData?.members);
  const reactionGroups = groupReactionsByEmoji(message.reactions, user?.id);

  const isDeleted = !!message.deleted_at;
  const isEdited = !!message.edited_at;
  const isOwnMessage = user?.id === message.user_id;
  const isPinned = !!message.pinned_at;
  const canPin = !!isAdmin;
  const canDelete = isOwnMessage || !!isAdmin;

  const handleReactionClick = (emoji: string, hasOwn: boolean) => {
    if (hasOwn) {
      removeReaction.mutate({ messageId: message.id, emoji });
    } else {
      addReaction.mutate({ messageId: message.id, emoji });
    }
  };

  const handleAddReaction = (emoji: string) => {
    addReaction.mutate({ messageId: message.id, emoji });
  };

  const handleStartEdit = () => {
    setIsEditing(true);
    setShowDropdown(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
  };

  const handleSaveEdit = (content: string) => {
    if (content.trim() && content.trim() !== message.content) {
      updateMessage.mutate({
        messageId: message.id,
        content: content.trim(),
      });
    }
    setIsEditing(false);
  };

  const handleDeleteClick = () => {
    setShowDropdown(false);
    setShowDeleteModal(true);
  };

  const handleCopyLink = () => {
    const url = `${window.location.origin}/workspaces/${workspaceId}/channels/${channelId}?msg=${message.id}`;
    navigator.clipboard.writeText(url);
    toast('Link copied to clipboard', 'success');
    setShowDropdown(false);
  };

  const handleTogglePin = () => {
    setShowDropdown(false);
    if (isPinned) {
      unpinMessage.mutate(message.id, {
        onError: () => toast('Failed to unpin message', 'error'),
      });
    } else {
      pinMessage.mutate(message.id, {
        onError: () => toast('Failed to pin message', 'error'),
      });
    }
  };

  const handleBlockUser = async () => {
    if (!message.user_id) return;
    try {
      await blockUser.mutateAsync(message.user_id);
      toast('User blocked', 'success');
    } catch {
      toast('Failed to block user', 'error');
    }
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

  // Auto-focus editor when entering edit mode
  useEffect(() => {
    if (isEditing) {
      const timer = setTimeout(() => editEditorRef.current?.focus(), 50);
      return () => clearTimeout(timer);
    }
  }, [isEditing]);

  const workspaceMembers = useEditorMembers(membersData?.members);
  const workspaceChannels = useEditorChannels(channels);

  // Deleted messages without replies: render nothing
  if (isDeleted && message.reply_count === 0) {
    return null;
  }

  // Deleted messages with replies: show placeholder with thread indicator
  if (isDeleted) {
    return (
      <div className="group px-4 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-800">
        <div className="flex items-start gap-3">
          {/* Trash icon in avatar-sized circle */}
          <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gray-200 dark:bg-gray-700">
            <TrashIcon className="h-4 w-4 text-gray-400 dark:text-gray-500" />
          </div>

          {/* Content */}
          <div className="min-w-0 flex-1">
            <span className="text-sm text-gray-400 italic dark:text-gray-500">
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
      id={`message-${message.id}`}
      ref={messageRef}
      className={cn(
        'group relative px-4 py-1.5',
        isDeleting
          ? '!max-h-0 overflow-hidden bg-red-400 !py-0 opacity-0 transition-all duration-500 dark:bg-red-700'
          : isEditing
            ? 'bg-yellow-50 dark:bg-yellow-900/10'
            : 'hover:bg-gray-100 dark:hover:bg-gray-800',
        (showDropdown || msgCtx.isOpen) &&
          !isDeleting &&
          !isEditing &&
          'bg-gray-100 dark:bg-gray-800',
      )}
      style={isDeleting ? { marginTop: 0, marginBottom: 0 } : undefined}
      onContextMenu={msgCtx.onContextMenu}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => {
        if (!showDropdown && !reactionPickerOpen && !msgCtx.isOpen) {
          setShowActions(false);
        }
      }}
    >
      <div className="flex items-start gap-3">
        {/* Avatar */}
        <Avatar
          src={message.user_avatar_url}
          gravatarSrc={message.user_gravatar_url}
          name={message.user_display_name || 'Unknown'}
          id={message.user_id}
          size="md"
          onClick={message.user_id ? () => openProfile(message.user_id!) : undefined}
          onContextMenu={message.user_id ? onUserContextMenu : undefined}
        />

        {/* Content */}
        <div className="min-w-0 flex-1">
          {/* Header */}
          <div className="flex items-baseline gap-2">
            <ClickableName
              userId={message.user_id}
              displayName={message.user_display_name || 'Unknown User'}
              onContextMenu={message.user_id ? onUserContextMenu : undefined}
            />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
            {isEdited && <span className="text-xs text-gray-400 dark:text-gray-500">(edited)</span>}
            {isPinned && (
              <span className="inline-flex items-center gap-0.5 text-xs text-amber-600 dark:text-amber-400">
                <MapPinIcon className="h-3 w-3" />
                Pinned
              </span>
            )}
          </div>

          {/* Broadcast thread reply indicator */}
          {message.thread_parent_id && message.also_send_to_channel && !isEditing && (
            <button
              type="button"
              onClick={() => openThread(message.thread_parent_id!)}
              className="cursor-pointer text-xs text-blue-600 hover:underline dark:text-blue-400"
            >
              replied to a thread
            </button>
          )}

          {/* Message content */}
          {isEditing ? (
            <div className="mt-1">
              <LazyRichTextEditor
                ref={editEditorRef}
                initialContent={message.content}
                onSubmit={handleSaveEdit}
                onEscape={handleCancelEdit}
                workspaceMembers={workspaceMembers}
                workspaceChannels={workspaceChannels}
                customEmojis={customEmojis}
                showToolbar
                showActionRow
                isPending={updateMessage.isPending}
                placeholder="Edit message..."
                submitLabel="Save"
              />
            </div>
          ) : (
            <CollapsibleMessage>
              {message.content && (
                <div className="break-words whitespace-pre-wrap text-gray-800 dark:text-gray-200">
                  <MessageContent
                    content={message.content}
                    members={membersData?.members}
                    channels={channels}
                    customEmojiMap={customEmojiMap}
                  />
                </div>
              )}
              {message.attachments && message.attachments.length > 0 && (
                <AttachmentDisplay attachments={message.attachments} />
              )}
              {message.link_preview &&
                (message.link_preview.type === 'message' ? (
                  <MessagePreviewDisplay
                    preview={message.link_preview}
                    onDismiss={
                      isOwnMessage ? () => deleteLinkPreview.mutate(message.id) : undefined
                    }
                    members={membersData?.members}
                    channels={channels}
                    customEmojiMap={customEmojiMap}
                  />
                ) : (
                  <LinkPreviewDisplay
                    preview={message.link_preview}
                    onDismiss={
                      isOwnMessage ? () => deleteLinkPreview.mutate(message.id) : undefined
                    }
                  />
                ))}
            </CollapsibleMessage>
          )}

          {/* Reactions */}
          <ReactionsDisplay
            reactions={reactionGroups}
            memberNames={memberNames}
            onReactionClick={handleReactionClick}
            customEmojiMap={customEmojiMap}
          />

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
        <MessageActionBar
          reactionPickerOpen={reactionPickerOpen}
          onReactionPickerOpenChange={setReactionPickerOpen}
          onReactionSelect={handleAddReaction}
          onReplyClick={() => openThread(message.id)}
          onCopyLink={handleCopyLink}
          onMarkUnread={() => markUnread.mutate(message.id)}
          showDropdown={showDropdown}
          onDropdownChange={setShowDropdown}
          onEdit={isOwnMessage ? handleStartEdit : undefined}
          onDelete={canDelete ? handleDeleteClick : undefined}
          onPin={canPin ? handleTogglePin : undefined}
          isPinned={isPinned}
          customEmojis={customEmojis}
        />
      )}

      {/* Message context menu */}
      <ContextMenu
        isOpen={msgCtx.isOpen}
        onOpenChange={msgCtx.setIsOpen}
        position={msgCtx.position}
      >
        <MenuItem
          onAction={() => openThread(message.id)}
          icon={<ChatBubbleBottomCenterTextIcon className="h-4 w-4" />}
        >
          Reply in Thread
        </MenuItem>
        <MenuItem onAction={handleCopyLink} icon={<LinkIcon className="h-4 w-4" />}>
          Copy Link
        </MenuItem>
        <MenuItem
          onAction={() => markUnread.mutate(message.id)}
          icon={<EyeSlashIcon className="h-4 w-4" />}
        >
          Mark Unread
        </MenuItem>
        {canPin && (
          <>
            <MenuSeparator />
            <MenuItem onAction={handleTogglePin} icon={<MapPinIcon className="h-4 w-4" />}>
              {isPinned ? 'Unpin Message' : 'Pin Message'}
            </MenuItem>
          </>
        )}
        {(isOwnMessage || canDelete) && (
          <>
            <MenuSeparator />
            {isOwnMessage && (
              <MenuItem onAction={handleStartEdit} icon={<PencilSquareIcon className="h-4 w-4" />}>
                Edit Message
              </MenuItem>
            )}
            <MenuItem
              onAction={handleDeleteClick}
              variant="danger"
              icon={<TrashIcon className="h-4 w-4" />}
            >
              Delete Message
            </MenuItem>
          </>
        )}
      </ContextMenu>

      {/* User context menu (shared by avatar + name) */}
      {message.user_id && (
        <ContextMenu
          isOpen={userCtx.isOpen}
          onOpenChange={userCtx.setIsOpen}
          position={userCtx.position}
        >
          <MenuItem
            onAction={() => openProfile(message.user_id!)}
            icon={<UserIcon className="h-4 w-4" />}
          >
            View Profile
          </MenuItem>
          <MenuItem
            onAction={async () => {
              if (!workspaceId) return;
              try {
                const result = await createDM.mutateAsync({ user_ids: [message.user_id!] });
                navigate(`/workspaces/${workspaceId}/channels/${result.channel.id}`);
              } catch {
                toast('Failed to start conversation', 'error');
              }
            }}
            icon={<ChatBubbleLeftIcon className="h-4 w-4" />}
          >
            Send Message
          </MenuItem>
          {!isOwnMessage && (
            <>
              <MenuSeparator />
              <MenuItem
                onAction={handleBlockUser}
                variant="danger"
                icon={<NoSymbolIcon className="h-4 w-4" />}
              >
                Block User
              </MenuItem>
            </>
          )}
        </ContextMenu>
      )}

      {/* Delete confirmation modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="Delete message"
        size="sm"
      >
        <p className="mb-4 text-gray-600 dark:text-gray-300">
          Are you sure you want to delete this message? This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onPress={() => setShowDeleteModal(false)}>
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
