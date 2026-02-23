import { useNavigate, useParams } from 'react-router-dom';
import { LockClosedIcon, TrashIcon } from '@heroicons/react/24/outline';
import { Avatar } from '../ui';
import { MessageContent } from './MessageContent';
import { DismissPreviewButton } from './DismissPreviewButton';
import { formatRelativeTime } from '../../lib/utils';
import type {
  LinkPreview,
  WorkspaceMemberWithUser,
  ChannelWithMembership,
  CustomEmoji,
} from '@enzyme/api-client';

interface MessagePreviewDisplayProps {
  preview: LinkPreview;
  onDismiss?: () => void;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  customEmojiMap?: Map<string, CustomEmoji>;
}

export function MessagePreviewDisplay({
  preview,
  onDismiss,
  members,
  channels,
  customEmojiMap,
}: MessagePreviewDisplayProps) {
  const navigate = useNavigate();
  const { workspaceId } = useParams<{ workspaceId: string }>();

  const isInaccessible = preview.linked_channel_type === 'inaccessible';
  const isDeleted = preview.linked_channel_type === 'deleted';

  const handleClick = () => {
    if (isInaccessible || isDeleted || !preview.linked_channel_id || !preview.linked_message_id)
      return;
    navigate(
      `/workspaces/${workspaceId}/channels/${preview.linked_channel_id}?msg=${preview.linked_message_id}`,
    );
  };

  if (isInaccessible || isDeleted) {
    return (
      <div className="group/preview relative mt-2 max-w-lg">
        <div className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2.5 dark:border-gray-700 dark:bg-gray-800/50">
          {isInaccessible ? (
            <LockClosedIcon className="h-4 w-4 flex-shrink-0 text-gray-400 dark:text-gray-500" />
          ) : (
            <TrashIcon className="h-4 w-4 flex-shrink-0 text-gray-400 dark:text-gray-500" />
          )}
          <span className="text-sm text-gray-500 italic dark:text-gray-400">
            {isInaccessible ? 'Message from a private channel' : 'This message was deleted'}
          </span>
        </div>
        {onDismiss && <DismissPreviewButton onDismiss={onDismiss} label="Remove message preview" />}
      </div>
    );
  }

  return (
    <div className="group/preview relative mt-2 max-w-lg">
      <button
        type="button"
        onClick={handleClick}
        className="block w-full cursor-pointer overflow-hidden rounded-lg border border-gray-200 bg-white text-left dark:border-gray-700 dark:bg-gray-900"
      >
        <div className="flex">
          {/* Blue left border accent */}
          <div className="w-1 flex-shrink-0 bg-blue-500" />
          <div className="min-w-0 flex-1 px-3 py-2">
            {/* Header: avatar + author + channel + time */}
            <div className="flex items-center gap-1.5">
              <Avatar
                src={preview.message_author_avatar_url}
                gravatarSrc={preview.message_author_gravatar_url}
                name={preview.message_author_name || 'Unknown'}
                id={preview.message_author_id}
                size="xs"
              />
              <span className="truncate text-sm font-medium text-gray-900 dark:text-white">
                {preview.message_author_name || 'Unknown'}
              </span>
              {preview.linked_channel_name && (
                <>
                  <span className="text-xs text-gray-400 dark:text-gray-500">in</span>
                  <span className="truncate text-xs font-medium text-gray-500 dark:text-gray-400">
                    #{preview.linked_channel_name}
                  </span>
                </>
              )}
              {preview.message_created_at && (
                <span className="flex-shrink-0 text-xs text-gray-400 dark:text-gray-500">
                  {formatRelativeTime(preview.message_created_at)}
                </span>
              )}
            </div>
            {/* Message content */}
            {preview.message_content && (
              <div className="mt-1 line-clamp-3 text-sm break-words text-gray-700 dark:text-gray-300">
                <MessageContent
                  content={preview.message_content}
                  members={members}
                  channels={channels}
                  customEmojiMap={customEmojiMap}
                />
              </div>
            )}
          </div>
        </div>
      </button>
      {onDismiss && <DismissPreviewButton onDismiss={onDismiss} label="Remove message preview" />}
    </div>
  );
}
