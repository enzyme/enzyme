import { useState, useRef, useMemo, useCallback, type KeyboardEvent, type FormEvent } from 'react';
import { Button as AriaButton, DropZone, FileTrigger } from 'react-aria-components';
import {
  DocumentIcon,
  ExclamationCircleIcon,
  XMarkIcon,
  PaperClipIcon,
  PaperAirplaneIcon,
} from '@heroicons/react/24/outline';
import { useSendMessage, useTyping, useUploadFile } from '../../hooks';
import { usePresenceStore } from '../../stores/presenceStore';
import { cn } from '../../lib/utils';

interface MessageComposerProps {
  channelId: string;
  workspaceId: string;
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

export function MessageComposer({
  channelId,
  workspaceId,
  placeholder = 'Type a message...',
}: MessageComposerProps) {
  const [content, setContent] = useState('');
  const [pendingAttachments, setPendingAttachments] = useState<PendingAttachment[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const sendMessage = useSendMessage(channelId);
  const uploadFile = useUploadFile(channelId);
  const { onTyping, onStopTyping } = useTyping(workspaceId, channelId);

  const typingUsersMap = usePresenceStore((state) => state.typingUsers);
  const typingUsers = useMemo(() => {
    const now = Date.now();
    const typers = typingUsersMap.get(channelId) || [];
    return typers.filter((t) => t.expiresAt > now);
  }, [typingUsersMap, channelId]);

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

  const hasContent = content.trim() !== '';
  const hasAttachments = completedAttachmentIds.length > 0;
  const isUploading = pendingAttachments.some((a) => a.status === 'uploading');
  const canSend = (hasContent || hasAttachments) && !sendMessage.isPending && !isUploading;

  const handleSubmit = async (e?: FormEvent) => {
    e?.preventDefault();

    if (!canSend) return;

    try {
      await sendMessage.mutateAsync({
        content: content.trim() || undefined,
        attachment_ids: hasAttachments ? completedAttachmentIds : undefined,
      });
      setContent('');
      // Clear attachments and revoke preview URLs
      pendingAttachments.forEach((a) => {
        if (a.previewUrl) URL.revokeObjectURL(a.previewUrl);
      });
      setPendingAttachments([]);
      onStopTyping();

      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto';
      }
    } catch {
      // Error handling is done via toast in mutation
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
    onTyping();

    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
    }
  };

  const handleBlur = () => {
    onStopTyping();
  };

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 p-4">
      {/* Typing indicator */}
      {typingUsers.length > 0 && (
        <div className="text-sm text-gray-500 dark:text-gray-400 mb-2 flex items-center gap-2">
          <span className="flex gap-1">
            <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
            <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
            <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
          </span>
          <span>
            {typingUsers.map((u) => u.displayName).join(', ')}{' '}
            {typingUsers.length === 1 ? 'is' : 'are'} typing...
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
                  className="w-20 h-20 object-cover"
                />
              ) : (
                <div className="w-20 h-20 bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
                  <DocumentIcon className="w-8 h-8 text-gray-400" />
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
        <form onSubmit={handleSubmit}>
          <div className="border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 focus-within:ring-2 focus-within:ring-primary-500 focus-within:border-primary-500">
            {/* Text input */}
            <textarea
              ref={textareaRef}
              value={content}
              onChange={(e) => handleChange(e.target.value)}
              onKeyDown={handleKeyDown}
              onBlur={handleBlur}
              placeholder={placeholder}
              rows={1}
              className="w-full px-4 py-3 resize-none bg-transparent text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none max-h-32"
            />

            {/* Actions row */}
            <div className="flex items-center justify-between px-2 py-1">
              {/* Attachment button */}
              <FileTrigger
                acceptedFileTypes={['image/*', '.pdf', '.txt', '.doc', '.docx']}
                allowsMultiple
                onSelect={(files) => files && handleFilesSelected(Array.from(files))}
              >
                <AriaButton
                  className="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                  aria-label="Attach files"
                >
                  <PaperClipIcon className="w-4 h-4" />
                </AriaButton>
              </FileTrigger>

              {/* Send button */}
              <button
                type="submit"
                disabled={!canSend}
                className={cn(
                  'p-1 rounded transition-colors',
                  canSend
                    ? 'text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20'
                    : 'text-gray-400 cursor-not-allowed'
                )}
              >
                <PaperAirplaneIcon className="w-4 h-4" />
              </button>
            </div>
          </div>
        </form>
      </DropZone>
    </div>
  );
}
