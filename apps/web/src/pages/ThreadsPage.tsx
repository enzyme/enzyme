import { useParams } from 'react-router-dom';
import {
  HashtagIcon,
  LockClosedIcon,
  ChatBubbleLeftRightIcon,
  ChatBubbleLeftEllipsisIcon,
} from '@heroicons/react/24/outline';
import { useUserThreads } from '../hooks/useThreads';
import { useThreadPanel } from '../hooks/usePanel';
import { usePageTitle } from '../hooks';
import { Spinner, Avatar, UnstyledButton } from '../components/ui';
import { cn, formatRelativeTime } from '../lib/utils';
import type { ThreadMessage } from '@enzyme/api-client';

function ChannelIcon({ type }: { type: string }) {
  switch (type) {
    case 'private':
      return <LockClosedIcon className="h-4 w-4 text-gray-500" />;
    case 'dm':
    case 'group_dm':
      return <ChatBubbleLeftRightIcon className="h-4 w-4 text-gray-500" />;
    default:
      return <HashtagIcon className="h-4 w-4 text-gray-500" />;
  }
}

function ThreadItem({ thread, onOpen }: { thread: ThreadMessage; onOpen: (id: string) => void }) {
  return (
    <UnstyledButton
      onPress={() => onOpen(thread.id)}
      className="block w-full border-b border-gray-200 p-4 text-left last:border-b-0 hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
    >
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 pt-1">
          <ChannelIcon type={thread.channel_type} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="mb-1 flex items-center gap-2">
            <span className="text-xs text-gray-500">
              {thread.channel_type === 'dm' || thread.channel_type === 'group_dm'
                ? 'Direct Message'
                : `#${thread.channel_name}`}
            </span>
            {thread.has_new_replies && (
              <span className="h-2 w-2 flex-shrink-0 rounded-full bg-blue-500" />
            )}
          </div>
          <div className="flex items-start gap-2">
            <Avatar
              src={thread.user_avatar_url}
              gravatarSrc={thread.user_gravatar_url}
              name={thread.user_display_name || 'Unknown'}
              id={thread.user_id}
              size="sm"
            />
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline gap-2">
                <span
                  className={cn(
                    'text-sm font-medium text-gray-700 dark:text-gray-300',
                    thread.has_new_replies && 'font-bold',
                  )}
                >
                  {thread.user_display_name || 'Unknown'}
                </span>
              </div>
              <p className="line-clamp-2 text-sm text-gray-600 dark:text-gray-400">
                {thread.content}
              </p>
            </div>
          </div>
          <div className="mt-2 flex items-center gap-3 text-xs text-gray-500">
            <span>
              {thread.reply_count} {thread.reply_count === 1 ? 'reply' : 'replies'}
            </span>
            {thread.last_reply_at && (
              <span>Last reply {formatRelativeTime(thread.last_reply_at)}</span>
            )}
            {thread.thread_participants && thread.thread_participants.length > 0 && (
              <div className="flex -space-x-1">
                {thread.thread_participants.slice(0, 3).map((p) => (
                  <Avatar
                    key={p.user_id}
                    src={p.avatar_url}
                    gravatarSrc={p.gravatar_url}
                    name={p.display_name || ''}
                    id={p.user_id}
                    size="xs"
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </UnstyledButton>
  );
}

export function ThreadsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  usePageTitle('Threads');
  const { openThread } = useThreadPanel();
  const { data, isLoading, hasNextPage, fetchNextPage, isFetchingNextPage } = useUserThreads({
    workspaceId: workspaceId || '',
  });

  const allThreads = data?.pages.flatMap((page) => page.threads) || [];
  const unreadCount = data?.pages[0]?.unread_thread_count ?? 0;
  const hasThreads = allThreads.length > 0;

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col bg-white dark:bg-gray-900">
      <div className="border-b border-gray-200 p-4 dark:border-gray-700">
        <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Threads</h1>
        <p className="text-sm text-gray-500">
          {hasThreads
            ? `${allThreads.length} thread${allThreads.length === 1 ? '' : 's'}${unreadCount > 0 ? ` (${unreadCount} with new replies)` : ''}`
            : 'No threads yet'}
        </p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {!hasThreads ? (
          <div className="flex h-full flex-col items-center justify-center p-8 text-center">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-800">
              <ChatBubbleLeftEllipsisIcon className="h-8 w-8 text-gray-400 dark:text-gray-500" />
            </div>
            <h2 className="mb-2 text-lg font-medium text-gray-900 dark:text-gray-100">
              No threads yet
            </h2>
            <p className="max-w-xs text-sm text-gray-500">
              Threads you participate in will appear here. Reply to a message to start a thread.
            </p>
          </div>
        ) : (
          <>
            {allThreads.map((thread) => (
              <ThreadItem key={thread.id} thread={thread} onOpen={openThread} />
            ))}
            {hasNextPage && (
              <div className="p-4 text-center">
                <UnstyledButton
                  onPress={() => fetchNextPage()}
                  isDisabled={isFetchingNextPage}
                  className="text-sm text-blue-600 hover:underline disabled:opacity-50 dark:text-blue-400"
                >
                  {isFetchingNextPage ? 'Loading...' : 'Load more'}
                </UnstyledButton>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
