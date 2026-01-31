import { useState, useRef, useMemo, type KeyboardEvent, type FormEvent } from 'react';
import { useSendMessage, useTyping } from '../../hooks';
import { usePresenceStore } from '../../stores/presenceStore';
import { cn } from '../../lib/utils';

interface MessageComposerProps {
  channelId: string;
  workspaceId: string;
  placeholder?: string;
}

export function MessageComposer({
  channelId,
  workspaceId,
  placeholder = 'Type a message...',
}: MessageComposerProps) {
  const [content, setContent] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const sendMessage = useSendMessage(channelId);
  const { onTyping, onStopTyping } = useTyping(workspaceId, channelId);

  // Select the raw map and filter in useMemo to avoid infinite loops
  const typingUsersMap = usePresenceStore((state) => state.typingUsers);
  const typingUsers = useMemo(() => {
    const now = Date.now();
    const typers = typingUsersMap.get(channelId) || [];
    return typers.filter((t) => t.expiresAt > now);
  }, [typingUsersMap, channelId]);

  const handleSubmit = async (e?: FormEvent) => {
    e?.preventDefault();

    const trimmedContent = content.trim();
    if (!trimmedContent || sendMessage.isPending) return;

    try {
      await sendMessage.mutateAsync({ content: trimmedContent });
      setContent('');
      onStopTyping();

      // Reset textarea height
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

    // Auto-resize textarea
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

      <form onSubmit={handleSubmit} className="relative">
        <textarea
          ref={textareaRef}
          value={content}
          onChange={(e) => handleChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={handleBlur}
          placeholder={placeholder}
          rows={1}
          className={cn(
            'w-full px-4 py-3 pr-12 resize-none',
            'border border-gray-300 dark:border-gray-600 rounded-lg',
            'bg-white dark:bg-gray-800 text-gray-900 dark:text-white',
            'placeholder-gray-500 dark:placeholder-gray-400',
            'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500',
            'max-h-32'
          )}
        />

        <button
          type="submit"
          disabled={!content.trim() || sendMessage.isPending}
          className={cn(
            'absolute right-2 bottom-2 p-2 rounded-lg transition-colors',
            content.trim()
              ? 'text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20'
              : 'text-gray-400 cursor-not-allowed'
          )}
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
          </svg>
        </button>
      </form>

      <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        Press <kbd className="px-1 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs">Enter</kbd> to send,{' '}
        <kbd className="px-1 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs">Shift+Enter</kbd> for new line
      </p>
    </div>
  );
}
