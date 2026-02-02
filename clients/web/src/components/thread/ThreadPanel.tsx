import {
  useState,
  useRef,
  useCallback,
  type FormEvent,
} from "react";
import { useParams } from "react-router-dom";
import { useQueryClient, useMutation } from "@tanstack/react-query";
import {
  DropZone,
} from "react-aria-components";
import {
  XMarkIcon,
  FaceSmileIcon,
  DocumentIcon,
  BellIcon,
  BellSlashIcon,
} from "@heroicons/react/24/outline";
import { RichTextEditor, type RichTextEditorRef } from "../editor";
import {
  useThreadMessages,
  useSendThreadReply,
  useMessage,
  useAuth,
  useUploadFile,
  useThreadSubscription,
  useSubscribeToThread,
  useUnsubscribeFromThread,
  useWorkspaceMembers,
  useChannels,
} from "../../hooks";
import { useThreadPanel, useProfilePanel } from "../../hooks/usePanel";
import { Avatar, MessageSkeleton } from "../ui";
import { ReactionPicker } from "../message/ReactionPicker";
import { AttachmentDisplay } from "../message/AttachmentDisplay";
import { MessageContent } from "../message/MessageContent";
import { cn, formatTime } from "../../lib/utils";
import { messagesApi } from "../../api/messages";
import type { MessageWithUser, MessageListResult, WorkspaceMemberWithUser } from "@feather/api-client";

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

  // Thread subscription
  const { data: subscriptionData, isLoading: isLoadingSubscription } =
    useThreadSubscription(messageId);
  const subscribe = useSubscribeToThread();
  const unsubscribe = useUnsubscribeFromThread();

  const subscriptionStatus = subscriptionData?.status ?? "none";
  const isSubscribed = subscriptionStatus === "subscribed";

  const handleToggleSubscription = () => {
    if (isSubscribed) {
      unsubscribe.mutate(messageId);
    } else {
      subscribe.mutate(messageId);
    }
  };

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

  return (
    <div className="w-96 border-l border-gray-200 dark:border-gray-700 flex flex-col bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
        <h3 className="font-semibold text-gray-900 dark:text-white">Thread</h3>
        <div className="flex items-center gap-1">
          <button
            onClick={handleToggleSubscription}
            disabled={isLoadingSubscription || subscribe.isPending || unsubscribe.isPending}
            className={cn(
              "p-1 rounded transition-colors",
              isSubscribed
                ? "text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20"
                : "text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700",
              (isLoadingSubscription || subscribe.isPending || unsubscribe.isPending) &&
                "opacity-50 cursor-not-allowed"
            )}
            title={isSubscribed ? "Unsubscribe from thread" : "Subscribe to thread"}
          >
            {isSubscribed ? (
              <BellIcon className="w-4 h-4" />
            ) : (
              <BellSlashIcon className="w-4 h-4" />
            )}
          </button>
          <button
            onClick={closeThread}
            className="p-1 text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
          >
            <XMarkIcon className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Loading state for parent message */}
      {isLoadingParent && !cachedMessage && (
        <div className="p-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <MessageSkeleton />
        </div>
      )}

      {/* Error state for parent message */}
      {parentError && !parentMessage && (
        <div className="p-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Message not found or you don't have access.
          </p>
        </div>
      )}

      {/* Parent message */}
      {parentMessage && <ParentMessage message={parentMessage} members={membersData?.members} />}

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
                {isFetchingNextPage ? "Loading..." : "Load more replies"}
              </button>
            )}

            {threadMessages.map((message) => (
              <ThreadMessage
                key={message.id}
                message={message}
                parentMessageId={messageId}
                members={membersData?.members}
              />
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

function ParentMessage({ message, members }: { message: MessageWithUser; members?: WorkspaceMemberWithUser[] }) {
  const { openProfile } = useProfilePanel();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);
  const [localReactions, setLocalReactions] = useState(message.reactions || []);

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
      // Check if already reacted
      if (localReactions.some((r) => r.user_id === userId && r.emoji === emoji))
        return;
      // Optimistic local update
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
      // Also update messages cache for consistency
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
      // Optimistic local update
      setLocalReactions((prev) =>
        prev.filter((r) => !(r.user_id === userId && r.emoji === emoji)),
      );
      // Also update messages cache for consistency
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
          name={message.user_display_name || "Unknown"}
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
          </div>
          {message.content && (
            <div className="text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
              <MessageContent content={message.content} members={members} />
            </div>
          )}
          {message.attachments && message.attachments.length > 0 && (
            <AttachmentDisplay attachments={message.attachments} />
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
      <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
        {message.reply_count} {message.reply_count === 1 ? "reply" : "replies"}
      </div>

      {/* Action button */}
      {showActions && (
        <button
          onClick={() => setShowReactionPicker(!showReactionPicker)}
          className="absolute right-3 top-3 p-1 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded shadow-sm hover:bg-gray-100 dark:hover:bg-gray-600"
          title="Add reaction"
        >
          <FaceSmileIcon className="w-4 h-4 text-gray-500" />
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
  members?: WorkspaceMemberWithUser[];
}

function ThreadMessage({ message, parentMessageId, members }: ThreadMessageProps) {
  const { openProfile } = useProfilePanel();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);

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
          name={message.user_display_name || "Unknown"}
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
          </div>
          {message.content && (
            <div className="text-sm text-gray-800 dark:text-gray-200 break-words whitespace-pre-wrap">
              <MessageContent content={message.content} members={members} />
            </div>
          )}
          {message.attachments && message.attachments.length > 0 && (
            <AttachmentDisplay attachments={message.attachments} />
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

      {/* Action button */}
      {showActions && (
        <button
          onClick={() => setShowReactionPicker(!showReactionPicker)}
          className="absolute right-2 top-1 p-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded shadow-sm hover:bg-gray-100 dark:hover:bg-gray-700"
          title="Add reaction"
        >
          <FaceSmileIcon className="w-3.5 h-3.5 text-gray-500" />
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

interface PendingAttachment {
  id: string;
  file: File;
  previewUrl?: string;
  status: "pending" | "uploading" | "complete" | "error";
  uploadedId?: string;
}

const ACCEPTED_IMAGE_TYPES = [
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
];
const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

function ThreadComposer({ parentMessageId, channelId }: ThreadComposerProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const [pendingAttachments, setPendingAttachments] = useState<
    PendingAttachment[]
  >([]);
  const [isDragging, setIsDragging] = useState(false);
  const editorRef = useRef<RichTextEditorRef>(null);
  const sendReply = useSendThreadReply(parentMessageId, channelId);
  const uploadFile = useUploadFile(channelId);
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const { data: channelsData } = useChannels(workspaceId);

  const uploadAttachment = useCallback(
    async (attachment: PendingAttachment) => {
      setPendingAttachments((prev) =>
        prev.map((a) =>
          a.id === attachment.id ? { ...a, status: "uploading" } : a,
        ),
      );

      try {
        const result = await uploadFile.mutateAsync(attachment.file);
        setPendingAttachments((prev) =>
          prev.map((a) =>
            a.id === attachment.id
              ? { ...a, status: "complete", uploadedId: result.file.id }
              : a,
          ),
        );
      } catch {
        setPendingAttachments((prev) =>
          prev.map((a) =>
            a.id === attachment.id ? { ...a, status: "error" } : a,
          ),
        );
      }
    },
    [uploadFile],
  );

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      const validFiles = files.filter((file) => file.size <= MAX_FILE_SIZE);
      const newAttachments: PendingAttachment[] = validFiles.map((file) => {
        const isImage = ACCEPTED_IMAGE_TYPES.includes(file.type);
        return {
          id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
          file,
          previewUrl: isImage ? URL.createObjectURL(file) : undefined,
          status: "pending" as const,
        };
      });

      setPendingAttachments((prev) => [...prev, ...newAttachments]);
      newAttachments.forEach((attachment) => uploadAttachment(attachment));
    },
    [uploadAttachment],
  );

  const removeAttachment = useCallback((id: string) => {
    setPendingAttachments((prev) => {
      const attachment = prev.find((a) => a.id === id);
      if (attachment?.previewUrl) URL.revokeObjectURL(attachment.previewUrl);
      return prev.filter((a) => a.id !== id);
    });
  }, []);

  const completedAttachmentIds = pendingAttachments
    .filter((a) => a.status === "complete" && a.uploadedId)
    .map((a) => a.uploadedId!);

  const hasAttachments = completedAttachmentIds.length > 0;
  const isUploading = pendingAttachments.some((a) => a.status === "uploading");

  const handleSubmit = async (content: string) => {
    const hasContent = content.trim() !== "";
    const canSend =
      (hasContent || hasAttachments) && !sendReply.isPending && !isUploading;

    if (!canSend) return;

    try {
      await sendReply.mutateAsync({
        content: hasContent ? content : undefined,
        attachment_ids: hasAttachments ? completedAttachmentIds : undefined,
      });
      pendingAttachments.forEach((a) => {
        if (a.previewUrl) URL.revokeObjectURL(a.previewUrl);
      });
      setPendingAttachments([]);
    } catch {
      // Error handled by mutation
    }
  };

  const handleFormSubmit = (e: FormEvent) => {
    e.preventDefault();
    const content = editorRef.current?.getContent() || "";
    handleSubmit(content);
    editorRef.current?.clear();
  };

  // Convert workspace members to the format expected by RichTextEditor
  const workspaceMembers = membersData?.members.map((m) => ({
    user_id: m.user_id,
    display_name: m.display_name,
    avatar_url: m.avatar_url,
  })) || [];

  // Convert channels to the format expected by RichTextEditor
  const workspaceChannels = channelsData?.channels.map((c) => ({
    id: c.id,
    name: c.name,
    type: c.type as 'public' | 'private' | 'dm',
  })) || [];

  return (
    <div className="p-3 border-t border-gray-200 dark:border-gray-700">
      {/* Pending attachments preview */}
      {pendingAttachments.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-2">
          {pendingAttachments.map((attachment) => (
            <div
              key={attachment.id}
              className={cn(
                "relative group rounded border overflow-hidden",
                attachment.status === "error"
                  ? "border-red-300 dark:border-red-700"
                  : "border-gray-200 dark:border-gray-700",
              )}
            >
              {attachment.previewUrl ? (
                <img
                  src={attachment.previewUrl}
                  alt={attachment.file.name}
                  className="w-12 h-12 object-cover"
                />
              ) : (
                <div className="w-12 h-12 bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
                  <DocumentIcon className="w-5 h-5 text-gray-400" />
                </div>
              )}
              {attachment.status === "uploading" && (
                <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                </div>
              )}
              <button
                type="button"
                onClick={() => removeAttachment(attachment.id)}
                className="absolute top-0 right-0 p-0.5 bg-black/50 text-white rounded-bl opacity-0 group-hover:opacity-100"
              >
                <XMarkIcon className="w-3 h-3" />
              </button>
            </div>
          ))}
        </div>
      )}

      <DropZone
        onDropEnter={() => setIsDragging(true)}
        onDropExit={() => setIsDragging(false)}
        onDrop={async (e) => {
          setIsDragging(false);
          const files = await Promise.all(
            e.items.filter((i) => i.kind === "file").map((i) => i.getFile()),
          );
          handleFilesSelected(files.filter((f): f is File => f !== null));
        }}
        className={cn("rounded-lg", isDragging && "ring-2 ring-primary-500")}
      >
        <form onSubmit={handleFormSubmit}>
          <RichTextEditor
            ref={editorRef}
            placeholder="Reply..."
            onSubmit={handleSubmit}
            workspaceMembers={workspaceMembers}
            workspaceChannels={workspaceChannels}
            showToolbar={false}
            disabled={sendReply.isPending}
            isPending={sendReply.isPending || isUploading}
            onAttachmentClick={handleFilesSelected}
          />
        </form>
      </DropZone>
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
