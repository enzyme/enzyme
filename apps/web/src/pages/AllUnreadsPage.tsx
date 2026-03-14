import { useParams, Link } from 'react-router-dom';
import { HashtagIcon, LockClosedIcon, ChatBubbleLeftRightIcon } from '@heroicons/react/24/outline';
import { useAllUnreads } from '../hooks/useAllUnreads';
import { usePageTitle } from '../hooks';
import { Spinner, UnstyledButton } from '../components/ui';
import { formatRelativeTime } from '../lib/utils';
import type { UnreadMessage } from '@enzyme/api-client';

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

function UnreadMessageItem({
  message,
  workspaceId,
}: {
  message: UnreadMessage;
  workspaceId: string;
}) {
  return (
    <Link
      to={`/workspaces/${workspaceId}/channels/${message.channel_id}`}
      className="block border-b border-gray-200 p-4 last:border-b-0 hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
    >
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 pt-1">
          <ChannelIcon type={message.channel_type} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="mb-1 flex items-center gap-2">
            <span className="font-medium text-gray-900 dark:text-gray-100">
              {message.channel_type === 'dm' || message.channel_type === 'group_dm'
                ? message.user_display_name || 'Direct Message'
                : `#${message.channel_name}`}
            </span>
            <span className="text-xs text-gray-500">{formatRelativeTime(message.created_at)}</span>
          </div>
          <div className="flex items-start gap-2">
            {message.user_avatar_url ? (
              <img
                src={message.user_avatar_url}
                alt=""
                className="h-6 w-6 flex-shrink-0 rounded-full"
              />
            ) : (
              <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-gray-300 text-xs font-medium text-gray-600 dark:bg-gray-600 dark:text-gray-300">
                {(message.user_display_name || '?')[0].toUpperCase()}
              </div>
            )}
            <div className="min-w-0">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {message.user_display_name || 'Unknown'}
              </span>
              <p className="line-clamp-2 text-sm text-gray-600 dark:text-gray-400">
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
  usePageTitle('Unreads');
  const { data, isLoading, hasNextPage, fetchNextPage, isFetchingNextPage } = useAllUnreads({
    workspaceId: workspaceId || '',
  });

  const allMessages = data?.pages.flatMap((page) => page.messages) || [];
  const hasUnreads = allMessages.length > 0;

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
        <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">All Unreads</h1>
        <p className="text-sm text-gray-500">
          {hasUnreads
            ? `${allMessages.length} unread message${allMessages.length === 1 ? '' : 's'}`
            : 'No unread messages'}
        </p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {!hasUnreads ? (
          <div className="flex h-full flex-col items-center justify-center p-8 text-center">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
              <svg
                className="h-8 w-8 text-green-600 dark:text-green-400"
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
            <h2 className="mb-2 text-lg font-medium text-gray-900 dark:text-gray-100">
              You're all caught up!
            </h2>
            <p className="max-w-xs text-sm text-gray-500">
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
