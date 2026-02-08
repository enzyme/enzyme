import { useState, useRef, useEffect } from "react";
import { useParams } from "react-router-dom";
import { useQueryClient, useMutation } from "@tanstack/react-query";
import {
  XMarkIcon,
} from "@heroicons/react/24/outline";
import {
  useThreadMessages,
  useMessage,
  useAuth,
  useWorkspaceMembers,
  useChannels,
  useAutoFocusComposer,
} from "../../hooks";
import {
  useUpdateMessage,
  useDeleteMessage,
  useMarkMessageUnread,
} from "../../hooks/useMessages";
import { useThreadPanel, useProfilePanel } from "../../hooks/usePanel";
import { Avatar, MessageSkeleton, Modal, Button, toast } from "../ui";
import { ThreadNotificationButton } from "./ThreadNotificationButton";
import { MessageActionBar } from "../message/MessageActionBar";
import { AttachmentDisplay } from "../message/AttachmentDisplay";
import { MessageContent } from "../message/MessageContent";
import { MessageComposer, type MessageComposerRef } from "../message/MessageComposer";
import { cn, formatTime } from "../../lib/utils";
import { messagesApi } from "../../api/messages";
import { useMarkThreadRead } from "../../hooks/useThreads";
import type { MessageWithUser, MessageListResult, WorkspaceMemberWithUser, ChannelWithMembership } from "@feather/api-client";

function ClickableName({
  userId,
  displayName,
}: {
  userId?: string;
  displayName: string;
}) {
  const { openProfile } = useProfilePanel();

  if (!userId) {
    return (
      <span className="font-medium text-gray-900 dark:text-white">
        {displayName}
      </span>
    );
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
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const queryClient = useQueryClient();
  const { closeThread } = useThreadPanel();
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useThreadMessages(messageId);
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const { data: channelsData } = useChannels(workspaceId);
  const composerRef = useRef<MessageComposerRef>(null);
  const markThreadRead = useMarkThreadRead(workspaceId || '');

  // Try to get parent message from cache first
  const cachedMessage = getParentMessageFromCache(queryClient, messageId);

  // Fetch from API if not in cache (for deep links)
  const {
    data: fetchedData,
    isLoading: isLoadingParent,
    error: parentError,
  } = useMessage(cachedMessage ? undefined : messageId);

  // Use cached message if available, otherwise use fetched
  const parentMessage = cachedMessage || fetchedData?.message;

  // Flatten thread messages (already in chronological order from API)
  const threadMessages = data?.pages.flatMap((page) => page.messages) || [];

  // Focus composer and mark thread as read when thread opens
  useEffect(() => {
    const timer = setTimeout(() => composerRef.current?.focus(), 50);
    markThreadRead.mutate({ messageId });
    return () => clearTimeout(timer);
  }, [messageId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-focus for typing while thread is open
  useAutoFocusComposer(composerRef, true);

  return (
    <div className="w-96 border-l border-gray-200 dark:border-gray-700 flex flex-col bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
        <h3 className="font-semibold text-gray-900 dark:text-white">Thread</h3>
        <div className="flex items-center gap-1">
          <ThreadNotificationButton messageId={messageId} />
          <button
            onClick={closeThread}
            className="p-1 text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
          >
            <XMarkIcon className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Thread messages + composer */}
      <div className="flex-1 overflow-y-auto">
        {/* Loading state for parent message */}
        {isLoadingParent && !cachedMessage && (
          <div className="p-4">
            <MessageSkeleton />
          </div>
        )}

        {/* Error state for parent message */}
        {parentError && !parentMessage && (
          <div className="p-4">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Message not found or you don't have access.
            </p>
          </div>
        )}

        {/* Parent message */}
        {parentMessage && <div className="h-5" />}
        {parentMessage && <ParentMessage message={parentMessage} members={membersData?.members} channels={channelsData?.channels} />}

        {/* Spacer below parent when no replies */}
        {parentMessage && parentMessage.reply_count === 0 && <div className="h-4" />}

        {/* Replies divider */}
        {parentMessage && parentMessage.reply_count > 0 && (
          <div className="flex items-center gap-4 px-4 py-3">
            <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
            <span className="text-xs font-medium text-gray-500 dark:text-gray-400">
              {parentMessage.reply_count} {parentMessage.reply_count === 1 ? "reply" : "replies"}
            </span>
            <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
          </div>
        )}

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
                {isFetchingNextPage ? "Loading..." : "Load more replies"}
              </button>
            )}

            {threadMessages.map((message) => (
              <ThreadMessage
                key={message.id}
                message={message}
                parentMessageId={messageId}
                members={membersData?.members}
                channels={channelsData?.channels}
              />
            ))}

            {/* Spacer below last message */}
            {threadMessages.length > 0 && <div className="h-4" />}
          </>
        )}

        {/* Reply composer - flows after thread messages */}
        {parentMessage && workspaceId && (
          <MessageComposer
            ref={composerRef}
            channelId={parentMessage.channel_id}
            workspaceId={workspaceId}
            parentMessageId={messageId}
            variant="thread"
            placeholder="Reply..."
          />
        )}
      </div>
    </div>
  );
}

function ParentMessage({ message, members, channels }: { message: MessageWithUser; members?: WorkspaceMemberWithUser[]; channels?: ChannelWithMembership[] }) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { openProfile } = useProfilePanel();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState("");
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const editTextareaRef = useRef<HTMLTextAreaElement>(null);
  const [localReactions, setLocalReactions] = useState(message.reactions || []);
  const updateMessage = useUpdateMessage();
  const deleteMessage = useDeleteMessage();
  const markUnread = useMarkMessageUnread(workspaceId || "");

  const isOwnMessage = user?.id === message.user_id;
  const isEdited = !!message.edited_at;

  // Group reactions by emoji
  const reactionGroups = localReactions.reduce(
    (acc, reaction) => {
      if (!acc[reaction.emoji]) {
        acc[reaction.emoji] = {
          emoji: reaction.emoji,
          count: 0,
          userIds: [],
          hasOwn: false,
        };
      }
      acc[reaction.emoji].count++;
      acc[reaction.emoji].userIds.push(reaction.user_id);
      if (reaction.user_id === user?.id) {
        acc[reaction.emoji].hasOwn = true;
      }
      return acc;
    },
    {} as Record<
      string,
      { emoji: string; count: number; userIds: string[]; hasOwn: boolean }
    >,
  );

  const addReaction = useMutation({
    mutationFn: (emoji: string) => messagesApi.addReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id || "temp";
      if (localReactions.some((r) => r.user_id === userId && r.emoji === emoji))
        return;
      setLocalReactions((prev) => [
        ...prev,
        {
          id: "temp",
          message_id: message.id,
          user_id: userId,
          emoji,
          created_at: new Date().toISOString(),
        },
      ]);
      queryClient.setQueriesData(
        { queryKey: ["messages"] },
        (
          old:
            | { pages: MessageListResult[]; pageParams: (string | undefined)[] }
            | undefined,
        ) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== message.id) return msg;
                const reactions = msg.reactions || [];
                if (
                  reactions.some(
                    (r) => r.user_id === userId && r.emoji === emoji,
                  )
                )
                  return msg;
                return {
                  ...msg,
                  reactions: [
                    ...reactions,
                    {
                      id: "temp",
                      message_id: message.id,
                      user_id: userId,
                      emoji,
                      created_at: new Date().toISOString(),
                    },
                  ],
                };
              }),
            })),
          };
        },
      );
    },
  });

  const removeReaction = useMutation({
    mutationFn: (emoji: string) =>
      messagesApi.removeReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id;
      setLocalReactions((prev) =>
        prev.filter((r) => !(r.user_id === userId && r.emoji === emoji)),
      );
      queryClient.setQueriesData(
        { queryKey: ["messages"] },
        (
          old:
            | { pages: MessageListResult[]; pageParams: (string | undefined)[] }
            | undefined,
        ) => {
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
                  reactions: reactions.filter(
                    (r) => !(r.user_id === userId && r.emoji === emoji),
                  ),
                };
              }),
            })),
          };
        },
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

  const handleStartEdit = () => {
    setEditContent(message.content);
    setIsEditing(true);
    setShowDropdown(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditContent("");
  };

  const handleSaveEdit = () => {
    if (editContent.trim() && editContent.trim() !== message.content) {
      updateMessage.mutate({
        messageId: message.id,
        content: editContent.trim(),
      });
    }
    setIsEditing(false);
    setEditContent("");
  };

  const handleDeleteClick = () => {
    setShowDropdown(false);
    setShowDeleteModal(true);
  };

  const handleDeleteConfirm = () => {
    setShowDeleteModal(false);
    deleteMessage.mutate(message.id);
  };

  const handleCopyLink = () => {
    const url = `${window.location.origin}/workspaces/${workspaceId}/channels/${message.channel_id}?msg=${message.id}`;
    navigator.clipboard.writeText(url);
    toast("Link copied to clipboard", "success");
    setShowDropdown(false);
  };

  const handleEditKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Escape") {
      handleCancelEdit();
    } else if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSaveEdit();
    }
  };

  useEffect(() => {
    if (isEditing && editTextareaRef.current) {
      editTextareaRef.current.focus();
      editTextareaRef.current.selectionStart =
        editTextareaRef.current.value.length;
    }
  }, [isEditing]);

  return (
    <div
      className={cn(
        "px-4 py-1.5 relative group",
        "hover:bg-gray-50 dark:hover:bg-gray-800/50",
        showDropdown && "bg-gray-50 dark:bg-gray-800/50",
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
        <Avatar
          src={message.user_avatar_url}
          name={message.user_display_name || "Unknown"}
          id={message.user_id}
          size="md"
          onClick={
            message.user_id ? () => openProfile(message.user_id!) : undefined
          }
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <ClickableName
              userId={message.user_id}
              displayName={message.user_display_name || "Unknown User"}
            />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
            {isEdited && (
              <span className="text-xs text-gray-400 dark:text-gray-500">
                (edited)
              </span>
            )}
          </div>

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
                  {updateMessage.isPending ? "Saving..." : "Save"}
                </button>
                <button
                  onClick={handleCancelEdit}
                  className="px-3 py-1 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                >
                  Cancel
                </button>
                <span className="text-gray-500 dark:text-gray-400 text-xs">
                  Esc to cancel, Enter to save
                </span>
              </div>
            </div>
          ) : (
            <>
              {message.content && (
                <div className="text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
                  <MessageContent content={message.content} members={members} channels={channels} />
                </div>
              )}
              {message.attachments && message.attachments.length > 0 && (
                <AttachmentDisplay attachments={message.attachments} />
              )}
            </>
          )}

          {/* Reactions */}
          {Object.values(reactionGroups).length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {Object.values(reactionGroups).map(({ emoji, count, hasOwn }) => (
                <button
                  key={emoji}
                  onClick={() => handleReactionClick(emoji, hasOwn)}
                  className={cn(
                    "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-sm border transition-colors",
                    hasOwn
                      ? "bg-primary-100 dark:bg-primary-900/30 border-primary-300 dark:border-primary-700"
                      : "bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600",
                  )}
                >
                  <span>{emoji}</span>
                  <span className="text-xs text-gray-600 dark:text-gray-300">
                    {count}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Action bar */}
      {showActions && !isEditing && (
        <MessageActionBar
          showReactionPicker={showReactionPicker}
          onReactionPickerToggle={() => setShowReactionPicker(!showReactionPicker)}
          onReactionSelect={handleAddReaction}
          onCopyLink={handleCopyLink}
          onMarkUnread={() => markUnread.mutate(message.id)}
          showDropdown={showDropdown}
          onDropdownChange={setShowDropdown}
          onEdit={isOwnMessage ? handleStartEdit : undefined}
          onDelete={isOwnMessage ? handleDeleteClick : undefined}
        />
      )}

      {/* Delete confirmation modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="Delete message"
        size="sm"
      >
        <p className="text-gray-600 dark:text-gray-300 mb-4">
          Are you sure you want to delete this message? This action cannot be
          undone.
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

interface ThreadMessageProps {
  message: MessageWithUser;
  parentMessageId: string;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
}

function ThreadMessage({ message, parentMessageId, members, channels }: ThreadMessageProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { openProfile } = useProfilePanel();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState("");
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const editTextareaRef = useRef<HTMLTextAreaElement>(null);
  const updateMessage = useUpdateMessage();
  const deleteMessage = useDeleteMessage();
  const markUnread = useMarkMessageUnread(workspaceId || "");

  const isOwnMessage = user?.id === message.user_id;
  const isEdited = !!message.edited_at;

  // Group reactions by emoji
  const reactionGroups = (message.reactions || []).reduce(
    (acc, reaction) => {
      if (!acc[reaction.emoji]) {
        acc[reaction.emoji] = {
          emoji: reaction.emoji,
          count: 0,
          userIds: [],
          hasOwn: false,
        };
      }
      acc[reaction.emoji].count++;
      acc[reaction.emoji].userIds.push(reaction.user_id);
      if (reaction.user_id === user?.id) {
        acc[reaction.emoji].hasOwn = true;
      }
      return acc;
    },
    {} as Record<
      string,
      { emoji: string; count: number; userIds: string[]; hasOwn: boolean }
    >,
  );

  const addReaction = useMutation({
    mutationFn: (emoji: string) => messagesApi.addReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id || "temp";
      // Update thread cache
      queryClient.setQueryData(
        ["thread", parentMessageId],
        (
          old:
            | { pages: MessageListResult[]; pageParams: (string | undefined)[] }
            | undefined,
        ) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              messages: page.messages.map((msg) => {
                if (msg.id !== message.id) return msg;
                const reactions = msg.reactions || [];
                if (
                  reactions.some(
                    (r) => r.user_id === userId && r.emoji === emoji,
                  )
                )
                  return msg;
                return {
                  ...msg,
                  reactions: [
                    ...reactions,
                    {
                      id: "temp",
                      message_id: message.id,
                      user_id: userId,
                      emoji,
                      created_at: new Date().toISOString(),
                    },
                  ],
                };
              }),
            })),
          };
        },
      );
    },
  });

  const removeReaction = useMutation({
    mutationFn: (emoji: string) =>
      messagesApi.removeReaction(message.id, emoji),
    onMutate: async (emoji) => {
      const userId = user?.id;
      queryClient.setQueryData(
        ["thread", parentMessageId],
        (
          old:
            | { pages: MessageListResult[]; pageParams: (string | undefined)[] }
            | undefined,
        ) => {
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
                  reactions: reactions.filter(
                    (r) => !(r.user_id === userId && r.emoji === emoji),
                  ),
                };
              }),
            })),
          };
        },
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

  const handleStartEdit = () => {
    setEditContent(message.content);
    setIsEditing(true);
    setShowDropdown(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditContent("");
  };

  const handleSaveEdit = () => {
    if (editContent.trim() && editContent.trim() !== message.content) {
      updateMessage.mutate({
        messageId: message.id,
        content: editContent.trim(),
      });
    }
    setIsEditing(false);
    setEditContent("");
  };

  const handleDeleteClick = () => {
    setShowDropdown(false);
    setShowDeleteModal(true);
  };

  const handleDeleteConfirm = () => {
    setShowDeleteModal(false);
    deleteMessage.mutate(message.id);
  };

  const handleCopyLink = () => {
    const channelId = message.channel_id;
    const url = `${window.location.origin}/workspaces/${workspaceId}/channels/${channelId}?msg=${message.id}`;
    navigator.clipboard.writeText(url);
    toast("Link copied to clipboard", "success");
    setShowDropdown(false);
  };

  const handleEditKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Escape") {
      handleCancelEdit();
    } else if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSaveEdit();
    }
  };

  // Auto-focus textarea when entering edit mode
  useEffect(() => {
    if (isEditing && editTextareaRef.current) {
      editTextareaRef.current.focus();
      editTextareaRef.current.selectionStart =
        editTextareaRef.current.value.length;
    }
  }, [isEditing]);

  return (
    <div
      className={cn(
        "px-4 py-2 relative group",
        "hover:bg-gray-50 dark:hover:bg-gray-800/50",
        showDropdown && "bg-gray-50 dark:bg-gray-800/50",
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
        <Avatar
          src={message.user_avatar_url}
          name={message.user_display_name || "Unknown"}
          id={message.user_id}
          size="sm"
          onClick={
            message.user_id ? () => openProfile(message.user_id!) : undefined
          }
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <ClickableName
              userId={message.user_id}
              displayName={message.user_display_name || "Unknown User"}
            />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
            {isEdited && (
              <span className="text-xs text-gray-400 dark:text-gray-500">
                (edited)
              </span>
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
                className="w-full p-2 border border-gray-300 dark:border-gray-600 rounded-lg resize-none bg-white dark:bg-gray-700 text-sm text-gray-800 dark:text-gray-200 focus:outline-none focus:ring-2 focus:ring-primary-500"
                rows={3}
              />
              <div className="flex items-center gap-2 text-sm">
                <button
                  onClick={handleSaveEdit}
                  disabled={updateMessage.isPending || !editContent.trim()}
                  className="px-3 py-1 bg-primary-600 text-white rounded hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-xs"
                >
                  {updateMessage.isPending ? "Saving..." : "Save"}
                </button>
                <button
                  onClick={handleCancelEdit}
                  className="px-3 py-1 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-xs"
                >
                  Cancel
                </button>
                <span className="text-gray-500 dark:text-gray-400 text-xs">
                  Esc to cancel, Enter to save
                </span>
              </div>
            </div>
          ) : (
            <>
              {message.content && (
                <div className="text-sm text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
                  <MessageContent content={message.content} members={members} channels={channels} />
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
              {Object.values(reactionGroups).map(({ emoji, count, hasOwn }) => (
                <button
                  key={emoji}
                  onClick={() => handleReactionClick(emoji, hasOwn)}
                  className={cn(
                    "inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-xs border transition-colors",
                    hasOwn
                      ? "bg-primary-100 dark:bg-primary-900/30 border-primary-300 dark:border-primary-700"
                      : "bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600",
                  )}
                >
                  <span>{emoji}</span>
                  <span className="text-gray-600 dark:text-gray-300">
                    {count}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Action bar */}
      {showActions && !isEditing && (
        <MessageActionBar
          showReactionPicker={showReactionPicker}
          onReactionPickerToggle={() => setShowReactionPicker(!showReactionPicker)}
          onReactionSelect={handleAddReaction}
          onCopyLink={handleCopyLink}
          onMarkUnread={() => markUnread.mutate(message.id)}
          showDropdown={showDropdown}
          onDropdownChange={setShowDropdown}
          onEdit={isOwnMessage ? handleStartEdit : undefined}
          onDelete={isOwnMessage ? handleDeleteClick : undefined}
        />
      )}

      {/* Delete confirmation modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="Delete message"
        size="sm"
      >
        <p className="text-gray-600 dark:text-gray-300 mb-4">
          Are you sure you want to delete this message? This action cannot be
          undone.
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

function getParentMessageFromCache(
  queryClient: ReturnType<typeof useQueryClient>,
  messageId: string,
): MessageWithUser | undefined {
  // Search through all message queries
  const queries = queryClient.getQueriesData<{
    pages: MessageListResult[];
    pageParams: (string | undefined)[];
  }>({ queryKey: ["messages"] });

  for (const [, data] of queries) {
    if (!data) continue;
    for (const page of data.pages) {
      const found = page.messages.find((m) => m.id === messageId);
      if (found) return found;
    }
  }

  return undefined;
}
