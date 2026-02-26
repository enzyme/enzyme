import { useState } from 'react';
import { DialogTrigger, Popover, Dialog } from 'react-aria-components';
import { MapPinIcon } from '@heroicons/react/24/outline';
import { IconButton, Tooltip, Spinner } from '../ui';
import { MessageContent } from '../message/MessageContent';
import { usePinnedMessages } from '../../hooks/useModeration';
import { formatTime } from '../../lib/utils';

interface PinnedMessagesPopoverProps {
  channelId: string;
}

export function PinnedMessagesPopover({ channelId }: PinnedMessagesPopoverProps) {
  const [isOpen, setIsOpen] = useState(false);
  const { data, isLoading } = usePinnedMessages(isOpen ? channelId : undefined);
  const pinnedCount = data?.messages?.length ?? 0;

  return (
    <DialogTrigger isOpen={isOpen} onOpenChange={setIsOpen}>
      <Tooltip content="Pinned messages">
        <IconButton
          aria-label="Pinned messages"
          className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        >
          <MapPinIcon className="h-4 w-4" />
        </IconButton>
      </Tooltip>
      <Popover
        placement="bottom end"
        className="w-96 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-900"
      >
        <Dialog className="outline-none">
          <div className="border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <h3 className="font-semibold text-gray-900 dark:text-white">
              Pinned Messages {pinnedCount > 0 && `(${pinnedCount})`}
            </h3>
          </div>
          <div className="max-h-96 overflow-y-auto">
            {isLoading ? (
              <div className="flex justify-center py-8">
                <Spinner size="md" />
              </div>
            ) : pinnedCount === 0 ? (
              <div className="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                No pinned messages in this channel yet.
              </div>
            ) : (
              <div className="divide-y divide-gray-100 dark:divide-gray-800">
                {data!.messages.map((msg) => (
                  <div
                    key={msg.id}
                    className="px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/50"
                  >
                    <div className="flex items-baseline gap-2">
                      <span className="text-sm font-medium text-gray-900 dark:text-white">
                        {msg.user_display_name || 'Unknown'}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {formatTime(msg.created_at)}
                      </span>
                    </div>
                    {msg.content && (
                      <div className="mt-1 line-clamp-3 text-sm text-gray-700 dark:text-gray-300">
                        <MessageContent content={msg.content} />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </Dialog>
      </Popover>
    </DialogTrigger>
  );
}
