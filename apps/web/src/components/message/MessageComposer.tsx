import {
  useState,
  useRef,
  useCallback,
  forwardRef,
  useImperativeHandle,
  type FormEvent,
} from 'react';
import { DropZone } from 'react-aria-components';
import { DocumentIcon, ExclamationCircleIcon, XMarkIcon } from '@heroicons/react/24/outline';
import {
  useSendMessage,
  useSendThreadReply,
  useTyping,
  useUploadFile,
  useAuth,
  useWorkspaceMembers,
  useChannels,
} from '../../hooks';
import { useScheduleMessage } from '../../hooks/useScheduledMessages';
import { useCustomEmojis } from '../../hooks/useCustomEmojis';
import { useTypingUsers } from '../../lib/presenceStore';
import { cn } from '../../lib/utils';
import {
  LazyRichTextEditor,
  useEditorMembers,
  useEditorChannels,
  type RichTextEditorRef,
} from '../editor';
import { AddEmojiModal } from '../editor/AddEmojiModal';
import { ScheduleMessageModal } from './ScheduleMessageModal';
import { toast } from '../ui';

export interface MessageComposerRef {
  focus: () => void;
  insertText: (text: string) => void;
}

interface MessageComposerProps {
  channelId: string;
  workspaceId: string;
  parentMessageId?: string;
  variant?: 'channel' | 'thread';
  placeholder?: string;
  channelName?: string;
  channelType?: string;
}

interface PendingAttachment {
  id: string;
  file: File;
  previewUrl?: string;
  status: 'pending' | 'uploading' | 'complete' | 'error';
  uploadedId?: string;
  progress?: number;
  error?: string;
}

const ACCEPTED_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];
const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

export const MessageComposer = forwardRef<MessageComposerRef, MessageComposerProps>(
  function MessageComposer(
    {
      channelId,
      workspaceId,
      parentMessageId,
      variant = 'channel',
      placeholder = 'Type a message...',
      channelName,
      channelType,
    },
    ref,
  ) {
    const [pendingAttachments, setPendingAttachments] = useState<PendingAttachment[]>([]);
    const [alsoSendToChannel, setAlsoSendToChannel] = useState(false);
    const [isDragging, setIsDragging] = useState(false);
    const [addEmojiOpen, setAddEmojiOpen] = useState(false);
    const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
    const editorRef = useRef<RichTextEditorRef>(null);

    useImperativeHandle(ref, () => ({
      focus: () => editorRef.current?.focus(),
      insertText: (text: string) => editorRef.current?.insertText(text),
    }));
    const sendMessage = useSendMessage(channelId);
    const sendThreadReply = useSendThreadReply(parentMessageId ?? '', channelId);
    const scheduleMessage = useScheduleMessage(channelId);
    const uploadFile = useUploadFile(channelId);
    const { onTyping, onStopTyping } = useTyping(workspaceId, channelId);
    const { user } = useAuth();
    const { data: membersData } = useWorkspaceMembers(workspaceId);
    const { data: channelsData } = useChannels(workspaceId);
    const { data: customEmojis } = useCustomEmojis(workspaceId);

    const isThreadVariant = variant === 'thread';
    const activeMutation = parentMessageId ? sendThreadReply : sendMessage;

    const typingUsers = useTypingUsers(channelId);
    const otherTypingUsers = typingUsers.filter((u) => u.userId !== user?.id);

    const uploadAttachment = useCallback(
      async (attachment: PendingAttachment) => {
        setPendingAttachments((prev) =>
          prev.map((a) => (a.id === attachment.id ? { ...a, status: 'uploading' } : a)),
        );

        try {
          const result = await uploadFile.mutateAsync(attachment.file);
          setPendingAttachments((prev) =>
            prev.map((a) =>
              a.id === attachment.id ? { ...a, status: 'complete', uploadedId: result.file.id } : a,
            ),
          );
        } catch {
          setPendingAttachments((prev) =>
            prev.map((a) =>
              a.id === attachment.id ? { ...a, status: 'error', error: 'Upload failed' } : a,
            ),
          );
        }
      },
      [uploadFile],
    );

    const handleFilesSelected = useCallback(
      (files: File[]) => {
        const validFiles = files.filter((file) => {
          if (file.size > MAX_FILE_SIZE) {
            return false;
          }
          return true;
        });

        const newAttachments: PendingAttachment[] = validFiles.map((file) => {
          const isImage = ACCEPTED_IMAGE_TYPES.includes(file.type);
          return {
            id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            file,
            previewUrl: isImage ? URL.createObjectURL(file) : undefined,
            status: 'pending' as const,
          };
        });

        setPendingAttachments((prev) => [...prev, ...newAttachments]);

        // Start uploading each file
        newAttachments.forEach((attachment) => {
          uploadAttachment(attachment);
        });
      },
      [uploadAttachment],
    );

    const removeAttachment = useCallback((id: string) => {
      setPendingAttachments((prev) => {
        const attachment = prev.find((a) => a.id === id);
        if (attachment?.previewUrl) {
          URL.revokeObjectURL(attachment.previewUrl);
        }
        return prev.filter((a) => a.id !== id);
      });
    }, []);

    const completedAttachmentIds = pendingAttachments
      .filter((a) => a.status === 'complete' && a.uploadedId)
      .map((a) => a.uploadedId!);

    const isUploading = pendingAttachments.some((a) => a.status === 'uploading');
    const hasAttachments = completedAttachmentIds.length > 0;

    const handleSubmit = async (content: string) => {
      const hasContent = content.trim() !== '';
      const canSend = (hasContent || hasAttachments) && !activeMutation.isPending && !isUploading;

      if (!canSend) return;

      try {
        await activeMutation.mutateAsync({
          content: hasContent ? content : undefined,
          attachment_ids: hasAttachments ? completedAttachmentIds : undefined,
          ...(isThreadVariant && alsoSendToChannel ? { also_send_to_channel: true } : {}),
        });

        // Clear attachments and revoke preview URLs
        pendingAttachments.forEach((a) => {
          if (a.previewUrl) URL.revokeObjectURL(a.previewUrl);
        });
        setPendingAttachments([]);
        setAlsoSendToChannel(false);
        onStopTyping();
      } catch {
        // Error handling is done via toast in mutation
      }
    };

    const handleFormSubmit = (e: FormEvent) => {
      e.preventDefault();
      const content = editorRef.current?.getContent() || '';
      handleSubmit(content);
      editorRef.current?.clear();
    };

    const handleSchedule = async (scheduledFor: string) => {
      const content = editorRef.current?.getContent() || '';
      if (!content.trim()) return;

      try {
        await scheduleMessage.mutateAsync({
          content: content.trim(),
          scheduled_for: scheduledFor,
          ...(parentMessageId ? { thread_parent_id: parentMessageId } : {}),
          ...(isThreadVariant && alsoSendToChannel ? { also_send_to_channel: true } : {}),
          ...(hasAttachments ? { attachment_ids: completedAttachmentIds } : {}),
        });

        editorRef.current?.clear();
        pendingAttachments.forEach((a) => {
          if (a.previewUrl) URL.revokeObjectURL(a.previewUrl);
        });
        setPendingAttachments([]);
        setAlsoSendToChannel(false);

        const schedDate = new Date(scheduledFor);
        toast(
          `Message scheduled for ${schedDate.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })} at ${schedDate.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', hour12: true })}`,
          'success',
        );
      } catch {
        // Error handled by mutation
      }
    };

    // Convert workspace members to the format expected by RichTextEditor
    const workspaceMembers = useEditorMembers(membersData?.members);
    const workspaceChannels = useEditorChannels(channelsData?.channels);

    // Attachment size classes based on variant
    const attachmentSizeClass = isThreadVariant ? 'w-12 h-12' : 'w-20 h-20';
    const attachmentIconClass = isThreadVariant ? 'w-5 h-5' : 'w-8 h-8';

    return (
      <div className="bg-white px-4 pb-4 dark:bg-gray-900">
        {/* Typing indicator - only show for channel variant */}
        {!isThreadVariant && otherTypingUsers.length > 0 && (
          <div className="mb-2 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <span className="flex gap-1">
              <span
                className="h-2 w-2 animate-bounce rounded-full bg-gray-400"
                style={{ animationDelay: '0ms' }}
              />
              <span
                className="h-2 w-2 animate-bounce rounded-full bg-gray-400"
                style={{ animationDelay: '150ms' }}
              />
              <span
                className="h-2 w-2 animate-bounce rounded-full bg-gray-400"
                style={{ animationDelay: '300ms' }}
              />
            </span>
            <span>
              {otherTypingUsers.map((u) => u.displayName).join(', ')}{' '}
              {otherTypingUsers.length === 1 ? 'is' : 'are'} typing...
            </span>
          </div>
        )}

        {/* Pending attachments preview */}
        {pendingAttachments.length > 0 && (
          <div className="mb-2 flex flex-wrap gap-2">
            {pendingAttachments.map((attachment) => (
              <div
                key={attachment.id}
                className={cn(
                  'group relative overflow-hidden rounded-lg border',
                  attachment.status === 'error'
                    ? 'border-red-300 dark:border-red-700'
                    : 'border-gray-200 dark:border-gray-700',
                )}
              >
                {attachment.previewUrl ? (
                  <img
                    src={attachment.previewUrl}
                    alt={attachment.file.name}
                    className={cn(attachmentSizeClass, 'object-cover')}
                  />
                ) : (
                  <div
                    className={cn(
                      attachmentSizeClass,
                      'flex items-center justify-center bg-gray-100 dark:bg-gray-800',
                    )}
                  >
                    <DocumentIcon className={cn(attachmentIconClass, 'text-gray-400')} />
                  </div>
                )}

                {/* Upload status overlay */}
                {attachment.status === 'uploading' && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                    <div className="h-6 w-6 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  </div>
                )}

                {attachment.status === 'error' && (
                  <div className="absolute inset-0 flex items-center justify-center bg-red-500/50">
                    <ExclamationCircleIcon className="h-6 w-6 text-white" />
                  </div>
                )}

                {/* Remove button */}
                <button
                  type="button"
                  onClick={() => removeAttachment(attachment.id)}
                  className="absolute top-0.5 right-0.5 cursor-pointer rounded bg-black/50 p-0.5 text-white opacity-0 transition-opacity group-hover:opacity-100 hover:bg-black/70"
                  title="Remove"
                >
                  <XMarkIcon className="h-4 w-4" />
                </button>

                {/* File name tooltip */}
                <div className="absolute right-0 bottom-0 left-0 truncate bg-black/50 px-1 py-0.5 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100">
                  {attachment.file.name}
                </div>
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
              e.items.filter((i) => i.kind === 'file').map((i) => i.getFile()),
            );
            handleFilesSelected(files.filter((f): f is File => f !== null));
          }}
          className={cn(
            'relative rounded-lg transition-colors',
            isDragging && 'bg-blue-100 ring-2 ring-blue-500 dark:bg-blue-900',
          )}
        >
          {/* Drop overlay */}
          {isDragging && (
            <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900">
              <span className="text-sm font-medium text-blue-600 dark:text-blue-400">
                Drop files to upload
              </span>
            </div>
          )}
          <form onSubmit={handleFormSubmit}>
            <LazyRichTextEditor
              ref={editorRef}
              placeholder={placeholder}
              onSubmit={handleSubmit}
              onTyping={isThreadVariant ? undefined : onTyping}
              onBlur={isThreadVariant ? undefined : onStopTyping}
              workspaceMembers={workspaceMembers}
              workspaceChannels={workspaceChannels}
              customEmojis={customEmojis}
              onAddEmoji={() => setAddEmojiOpen(true)}
              showToolbar
              disabled={activeMutation.isPending}
              isPending={activeMutation.isPending || isUploading}
              onAttachmentClick={handleFilesSelected}
              onScheduleClick={() => setScheduleModalOpen(true)}
              onSchedule={handleSchedule}
              belowEditor={
                isThreadVariant ? (
                  <label className="flex cursor-pointer items-center gap-2 px-4 py-1.5 select-none">
                    <input
                      type="checkbox"
                      checked={alsoSendToChannel}
                      onChange={(e) => setAlsoSendToChannel(e.target.checked)}
                      className="rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700"
                    />
                    <span className="text-xs text-gray-500 dark:text-gray-400">
                      Also send to{' '}
                      {channelType === 'dm' || channelType === 'group_dm'
                        ? 'direct message'
                        : channelName
                          ? `#${channelName}`
                          : 'channel'}
                    </span>
                  </label>
                ) : undefined
              }
            />
          </form>
        </DropZone>
        <AddEmojiModal
          isOpen={addEmojiOpen}
          onClose={() => setAddEmojiOpen(false)}
          workspaceId={workspaceId}
          customEmojis={customEmojis ?? []}
        />
        <ScheduleMessageModal
          isOpen={scheduleModalOpen}
          onClose={() => setScheduleModalOpen(false)}
          onSchedule={handleSchedule}
        />
      </div>
    );
  },
);
