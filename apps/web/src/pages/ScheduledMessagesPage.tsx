import { useState, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import {
  ClockIcon,
  ExclamationTriangleIcon,
  HashtagIcon,
  LockClosedIcon,
} from '@heroicons/react/24/outline';
import {
  useScheduledMessages,
  useDeleteScheduledMessage,
  useSendScheduledMessageNow,
  useRetryScheduledMessage,
} from '../hooks/useScheduledMessages';
import { EditScheduledMessageModal } from '../components/message/EditScheduledMessageModal';
import { Spinner, Button } from '../components/ui';
import type { ScheduledMessage } from '@enzyme/api-client';

function ChannelIcon({ type }: { type?: string }) {
  if (type === 'private') return <LockClosedIcon className="h-3.5 w-3.5" />;
  return <HashtagIcon className="h-3.5 w-3.5" />;
}

function ScheduledMessageItem({
  message,
  onEdit,
  onDelete,
  onSendNow,
  onRetry,
  isDeleting,
  isSending,
  isRetrying,
}: {
  message: ScheduledMessage;
  onEdit: () => void;
  onDelete: () => void;
  onSendNow: () => void;
  onRetry: () => void;
  isDeleting: boolean;
  isSending: boolean;
  isRetrying: boolean;
}) {
  const scheduledDate = new Date(message.scheduled_for);
  const isPast = scheduledDate < new Date();
  const isFailed = message.status === 'failed';

  return (
    <div className="border-b border-gray-200 p-4 last:border-b-0 dark:border-gray-700">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          {message.channel_name && (
            <div className="mb-1 flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
              <ChannelIcon />
              <span>{message.channel_name}</span>
            </div>
          )}
          <p className="mb-2 line-clamp-3 text-sm whitespace-pre-wrap text-gray-900 dark:text-gray-100">
            {message.content}
          </p>
          {isFailed && (
            <div className="mb-2 flex items-center gap-1.5 text-xs text-red-600 dark:text-red-400">
              <ExclamationTriangleIcon className="h-3.5 w-3.5 flex-shrink-0" />
              <span>Failed: {message.last_error || 'Unknown error'}</span>
            </div>
          )}
          <div className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
            <ClockIcon className="h-3.5 w-3.5" />
            <span className={isPast && !isFailed ? 'text-amber-600 dark:text-amber-400' : ''}>
              {isPast && !isFailed ? 'Overdue - ' : ''}
              {scheduledDate.toLocaleDateString('en-US', {
                weekday: 'short',
                month: 'short',
                day: 'numeric',
              })}{' '}
              at{' '}
              {scheduledDate.toLocaleTimeString('en-US', {
                hour: 'numeric',
                minute: '2-digit',
                hour12: true,
              })}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          {isFailed ? (
            <>
              <Button size="sm" variant="secondary" onPress={onRetry} isLoading={isRetrying}>
                Retry
              </Button>
              <Button size="sm" variant="secondary" onPress={onDelete} isLoading={isDeleting}>
                Delete
              </Button>
            </>
          ) : (
            <>
              <Button size="sm" variant="secondary" onPress={onEdit}>
                Edit
              </Button>
              <Button size="sm" variant="secondary" onPress={onSendNow} isLoading={isSending}>
                Send Now
              </Button>
              <Button size="sm" variant="secondary" onPress={onDelete} isLoading={isDeleting}>
                Cancel
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

export function ScheduledMessagesPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { data, isLoading } = useScheduledMessages(workspaceId || '');
  const deleteMutation = useDeleteScheduledMessage();
  const sendNowMutation = useSendScheduledMessageNow();
  const retryMutation = useRetryScheduledMessage();
  const [editingMessage, setEditingMessage] = useState<ScheduledMessage | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [sendingId, setSendingId] = useState<string | null>(null);
  const [retryingId, setRetryingId] = useState<string | null>(null);

  const messages = useMemo(() => data?.scheduled_messages || [], [data?.scheduled_messages]);

  // Group messages by channel
  const groupedMessages = useMemo(() => {
    const groups: Record<string, ScheduledMessage[]> = {};
    for (const msg of messages) {
      const key = msg.channel_name || msg.channel_id;
      if (!groups[key]) groups[key] = [];
      groups[key].push(msg);
    }
    return groups;
  }, [messages]);

  const handleDelete = (id: string) => {
    setDeletingId(id);
    deleteMutation.mutate(id, {
      onSettled: () => setDeletingId(null),
    });
  };

  const handleSendNow = (id: string) => {
    setSendingId(id);
    sendNowMutation.mutate(id, {
      onSettled: () => setSendingId(null),
    });
  };

  const handleRetry = (id: string) => {
    setRetryingId(id);
    retryMutation.mutate(id, {
      onSettled: () => setRetryingId(null),
    });
  };

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-gray-200 px-6 py-3 dark:border-gray-700">
        <ClockIcon className="h-5 w-5 text-gray-500" />
        <h1 className="text-lg font-semibold text-gray-900 dark:text-white">Scheduled Messages</h1>
        {messages.length > 0 && (
          <span className="rounded-full bg-gray-200 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
            {messages.length}
          </span>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-gray-500 dark:text-gray-400">
            <ClockIcon className="h-12 w-12 text-gray-300 dark:text-gray-600" />
            <p className="text-lg font-medium">No scheduled messages</p>
            <p className="text-sm">
              Use the schedule button in the message composer to schedule messages for later.
            </p>
          </div>
        ) : (
          Object.entries(groupedMessages).map(([channelName, msgs]) => (
            <div key={channelName}>
              {msgs.map((msg) => (
                <ScheduledMessageItem
                  key={msg.id}
                  message={msg}
                  onEdit={() => setEditingMessage(msg)}
                  onDelete={() => handleDelete(msg.id)}
                  onSendNow={() => handleSendNow(msg.id)}
                  onRetry={() => handleRetry(msg.id)}
                  isDeleting={deletingId === msg.id}
                  isSending={sendingId === msg.id}
                  isRetrying={retryingId === msg.id}
                />
              ))}
            </div>
          ))
        )}
      </div>

      <EditScheduledMessageModal
        isOpen={editingMessage !== null}
        onClose={() => setEditingMessage(null)}
        message={editingMessage}
      />
    </div>
  );
}
