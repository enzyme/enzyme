import { useState, useRef, useCallback, forwardRef, useImperativeHandle, type FormEvent } from 'react';
import { DropZone } from 'react-aria-components';
import {
  DocumentIcon,
  ExclamationCircleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { useSendMessage, useSendThreadReply, useTyping, useUploadFile, useAuth, useWorkspaceMembers, useChannels } from '../../hooks';
import { useCustomEmojis } from '../../hooks/useCustomEmojis';
import { useTypingUsers } from '../../lib/presenceStore';
import { cn } from '../../lib/utils';
import { RichTextEditor, type RichTextEditorRef } from '../editor';

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
    },
    ref
  ) {
  const [pendingAttachments, setPendingAttachments] = useState<PendingAttachment[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const editorRef = useRef<RichTextEditorRef>(null);

  useImperativeHandle(ref, () => ({
    focus: () => editorRef.current?.focus(),
    insertText: (text: string) => editorRef.current?.insertText(text),
  }));
  const sendMessage = useSendMessage(channelId);
  const sendThreadReply = useSendThreadReply(parentMessageId ?? '', channelId);
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

  const uploadAttachment = useCallback(async (attachment: PendingAttachment) => {
    setPendingAttachments((prev) =>
      prev.map((a) => (a.id === attachment.id ? { ...a, status: 'uploading' } : a))
    );

    try {
      const result = await uploadFile.mutateAsync(attachment.file);
      setPendingAttachments((prev) =>
        prev.map((a) =>
          a.id === attachment.id
            ? { ...a, status: 'complete', uploadedId: result.file.id }
            : a
        )
      );
    } catch {
      setPendingAttachments((prev) =>
        prev.map((a) =>
          a.id === attachment.id
            ? { ...a, status: 'error', error: 'Upload failed' }
            : a
        )
      );
    }
  }, [uploadFile]);

  const handleFilesSelected = useCallback((files: File[]) => {
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
  }, [uploadAttachment]);

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
      });

      // Clear attachments and revoke preview URLs
      pendingAttachments.forEach((a) => {
        if (a.previewUrl) URL.revokeObjectURL(a.previewUrl);
      });
      setPendingAttachments([]);
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

  // Attachment size classes based on variant
  const attachmentSizeClass = isThreadVariant ? 'w-12 h-12' : 'w-20 h-20';
  const attachmentIconClass = isThreadVariant ? 'w-5 h-5' : 'w-8 h-8';

  return (
    <div className="pb-4 px-4 bg-white dark:bg-gray-900">
      {/* Typing indicator - only show for channel variant */}
      {!isThreadVariant && otherTypingUsers.length > 0 && (
        <div className="text-sm text-gray-500 dark:text-gray-400 mb-2 flex items-center gap-2">
          <span className="flex gap-1">
            <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
            <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
            <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
          </span>
          <span>
            {otherTypingUsers.map((u) => u.displayName).join(', ')}{' '}
            {otherTypingUsers.length === 1 ? 'is' : 'are'} typing...
          </span>
        </div>
      )}

      {/* Pending attachments preview */}
      {pendingAttachments.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-2">
          {pendingAttachments.map((attachment) => (
            <div
              key={attachment.id}
              className={cn(
                'relative group rounded-lg border overflow-hidden',
                attachment.status === 'error'
                  ? 'border-red-300 dark:border-red-700'
                  : 'border-gray-200 dark:border-gray-700'
              )}
            >
              {attachment.previewUrl ? (
                <img
                  src={attachment.previewUrl}
                  alt={attachment.file.name}
                  className={cn(attachmentSizeClass, 'object-cover')}
                />
              ) : (
                <div className={cn(attachmentSizeClass, 'bg-gray-100 dark:bg-gray-800 flex items-center justify-center')}>
                  <DocumentIcon className={cn(attachmentIconClass, 'text-gray-400')} />
                </div>
              )}

              {/* Upload status overlay */}
              {attachment.status === 'uploading' && (
                <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
                  <div className="w-6 h-6 border-2 border-white border-t-transparent rounded-full animate-spin" />
                </div>
              )}

              {attachment.status === 'error' && (
                <div className="absolute inset-0 bg-red-500/50 flex items-center justify-center">
                  <ExclamationCircleIcon className="w-6 h-6 text-white" />
                </div>
              )}

              {/* Remove button */}
              <button
                type="button"
                onClick={() => removeAttachment(attachment.id)}
                className="absolute top-0.5 right-0.5 p-0.5 bg-black/50 hover:bg-black/70 text-white rounded opacity-0 group-hover:opacity-100 transition-opacity"
                title="Remove"
              >
                <XMarkIcon className="w-4 h-4" />
              </button>

              {/* File name tooltip */}
              <div className="absolute bottom-0 left-0 right-0 bg-black/50 text-white text-xs px-1 py-0.5 truncate opacity-0 group-hover:opacity-100 transition-opacity">
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
            e.items
              .filter((i) => i.kind === 'file')
              .map((i) => i.getFile())
          );
          handleFilesSelected(files.filter((f): f is File => f !== null));
        }}
        className={cn(
          'relative rounded-lg transition-colors',
          isDragging && 'ring-2 ring-primary-500 bg-primary-50 dark:bg-primary-900/20'
        )}
      >
        {/* Drop overlay */}
        {isDragging && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-primary-50/90 dark:bg-primary-900/80 rounded-lg pointer-events-none">
            <span className="text-sm font-medium text-primary-600 dark:text-primary-400">
              Drop files to upload
            </span>
          </div>
        )}
        <form onSubmit={handleFormSubmit}>
          <RichTextEditor
            ref={editorRef}
            placeholder={placeholder}
            onSubmit={handleSubmit}
            onTyping={isThreadVariant ? undefined : onTyping}
            onBlur={isThreadVariant ? undefined : onStopTyping}
            workspaceMembers={workspaceMembers}
            workspaceChannels={workspaceChannels}
            customEmojis={customEmojis}
            showToolbar
            disabled={activeMutation.isPending}
            isPending={activeMutation.isPending || isUploading}
            onAttachmentClick={handleFilesSelected}
          />
        </form>
      </DropZone>
    </div>
  );
});
