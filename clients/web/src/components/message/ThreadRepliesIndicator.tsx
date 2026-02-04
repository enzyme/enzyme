import { ChevronRightIcon } from '@heroicons/react/24/outline';
import { AvatarStack } from '../ui';
import { useThreadPanel } from '../../hooks/usePanel';
import { formatRelativeTime } from '../../lib/utils';
import type { ThreadParticipant } from '@feather/api-client';

interface ThreadRepliesIndicatorProps {
  messageId: string;
  replyCount: number;
  lastReplyAt?: string;
  threadParticipants?: ThreadParticipant[];
}

export function ThreadRepliesIndicator({
  messageId,
  replyCount,
  lastReplyAt,
  threadParticipants,
}: ThreadRepliesIndicatorProps) {
  const { openThread } = useThreadPanel();

  if (replyCount === 0) {
    return null;
  }

  return (
    <button
      onClick={() => openThread(messageId)}
      className="mt-2 flex items-center gap-2 group/thread hover:bg-white dark:hover:bg-gray-900 hover:border hover:border-gray-200 dark:hover:border-gray-700 rounded-lg px-2 py-1 -mx-2 border border-transparent min-w-[300px]"
    >
      {threadParticipants && threadParticipants.length > 0 && (
        <AvatarStack users={threadParticipants} showCount={false} />
      )}
      <span className="text-sm text-primary-600 dark:text-primary-400">
        {replyCount} {replyCount === 1 ? 'reply' : 'replies'}
      </span>
      {lastReplyAt && (
        <span className="text-xs text-gray-500 dark:text-gray-400">
          Last reply {formatRelativeTime(lastReplyAt)}
        </span>
      )}
      <ChevronRightIcon className="w-4 h-4 text-gray-400 dark:text-gray-500 opacity-0 group-hover/thread:opacity-100 ml-auto" />
    </button>
  );
}
