import { useParams, Link } from 'react-router-dom';
import { HashtagIcon, LockClosedIcon, ChatBubbleLeftRightIcon } from '@heroicons/react/24/outline';
import { useAllUnreads } from '../hooks/useAllUnreads';
import { Spinner } from '../components/ui';
import { formatRelativeTime } from '../lib/utils';
import type { UnreadMessage } from '@feather/api-client';

function ChannelIcon({ type }: { type: string }) {
  switch (type) {
    case 'private':
      return <LockClosedIcon className="h-4 w-4 text-zinc-500" />;
    case 'dm':
    case 'group_dm':
      return <ChatBubbleLeftRightIcon className="h-4 w-4 text-zinc-500" />;
    default:
      return <HashtagIcon className="h-4 w-4 text-zinc-500" />;
  }
}

function UnreadMessageItem({ message, workspaceId }: { message: UnreadMessage; workspaceId: string }) {
  return (
    <Link
      to={`/w/${workspaceId}/c/${message.channel_id}`}
      className="block p-4 hover:bg-zinc-50 dark:hover:bg-zinc-800/50 border-b border-zinc-200 dark:border-zinc-700 last:border-b-0"
    >
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 pt-1">
          <ChannelIcon type={message.channel_type} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 mb-1">
            <span className="font-medium text-zinc-900 dark:text-zinc-100">
              {message.channel_type === 'dm' || message.channel_type === 'group_dm'
                ? message.user_display_name || 'Direct Message'
                : `#${message.channel_name}`}
            </span>
            <span className="text-xs text-zinc-500">
              {formatRelativeTime(new Date(message.created_at))}
            </span>
          </div>
          <div className="flex items-start gap-2">
            {message.user_avatar_url ? (
              <img
                src={message.user_avatar_url}
                alt=""
                className="h-6 w-6 rounded-full flex-shrink-0"
              />
            ) : (
              <div className="h-6 w-6 rounded-full bg-zinc-300 dark:bg-zinc-600 flex-shrink-0 flex items-center justify-center text-xs font-medium text-zinc-600 dark:text-zinc-300">
                {(message.user_display_name || '?')[0].toUpperCase()}
              </div>
            )}
            <div className="min-w-0">
              <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                {message.user_display_name || 'Unknown'}
              </span>
              <p className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2">
                {message.content}
              </p>
            </div>
          </div>
        </div>
      </div>
    </Link>
  );
}

export function AllUnreadsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { data, isLoading, hasNextPage, fetchNextPage, isFetchingNextPage } = useAllUnreads({
    workspaceId: workspaceId || '',
  });

  const allMessages = data?.pages.flatMap((page) => page.messages) || [];
  const hasUnreads = allMessages.length > 0;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full bg-white dark:bg-zinc-900">
      <div className="border-b border-zinc-200 dark:border-zinc-700 p-4">
        <h1 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          All Unreads
        </h1>
        <p className="text-sm text-zinc-500">
          {hasUnreads
            ? `${allMessages.length} unread message${allMessages.length === 1 ? '' : 's'}`
            : 'No unread messages'}
        </p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {!hasUnreads ? (
          <div className="flex flex-col items-center justify-center h-full text-center p-8">
            <div className="w-16 h-16 mb-4 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
              <svg
                className="w-8 h-8 text-green-600 dark:text-green-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
            <h2 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-2">
              You're all caught up!
            </h2>
            <p className="text-sm text-zinc-500 max-w-xs">
              No unread messages in any of your channels. New messages will appear here.
            </p>
          </div>
        ) : (
          <>
            {allMessages.map((message) => (
              <UnreadMessageItem
                key={message.id}
                message={message}
                workspaceId={workspaceId || ''}
              />
            ))}
            {hasNextPage && (
              <div className="p-4 text-center">
                <button
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="text-sm text-blue-600 dark:text-blue-400 hover:underline disabled:opacity-50"
                >
                  {isFetchingNextPage ? 'Loading...' : 'Load more'}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
