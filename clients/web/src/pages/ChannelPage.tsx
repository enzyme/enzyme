import { useState, useRef, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useChannels, useArchiveChannel } from '../hooks';
import { useMarkChannelAsRead } from '../hooks/useChannels';
import { MessageList, MessageComposer } from '../components/message';
import { Spinner, Modal, Button, toast } from '../components/ui';
import { getChannelIcon } from '../lib/utils';

export function ChannelPage() {
  const { workspaceId, channelId } = useParams<{
    workspaceId: string;
    channelId: string;
  }>();
  const navigate = useNavigate();

  const { data: channelsData, isLoading } = useChannels(workspaceId);
  const channel = channelsData?.channels.find((c) => c.id === channelId);
  const archiveChannel = useArchiveChannel(workspaceId || '');
  const markAsRead = useMarkChannelAsRead(workspaceId || '');

  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isArchiveModalOpen, setIsArchiveModalOpen] = useState(false);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const menuRef = useRef<HTMLDivElement>(null);
  const markAsReadTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Auto mark-as-read when user is at bottom for 2 seconds
  useEffect(() => {
    if (!channelId || !isAtBottom || !channel || channel.unread_count === 0) {
      if (markAsReadTimerRef.current) {
        clearTimeout(markAsReadTimerRef.current);
        markAsReadTimerRef.current = null;
      }
      return;
    }

    const currentChannelId = channelId;
    markAsReadTimerRef.current = setTimeout(() => {
      markAsRead.mutate({ channelId: currentChannelId });
    }, 2000);

    return () => {
      if (markAsReadTimerRef.current) {
        clearTimeout(markAsReadTimerRef.current);
      }
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [channelId, isAtBottom, channel?.unread_count]);

  // Reset state when changing channels
  useEffect(() => {
    setIsAtBottom(true);
  }, [channelId]);

  const handleAtBottomChange = useCallback((atBottom: boolean) => {
    setIsAtBottom(atBottom);
  }, []);

  // Close menu when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsMenuOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleArchive = async () => {
    if (!channelId || !workspaceId) return;
    try {
      await archiveChannel.mutateAsync(channelId);
      toast('Channel archived', 'success');
      setIsArchiveModalOpen(false);
      navigate(`/workspaces/${workspaceId}`);
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to archive channel', 'error');
    }
  };

  const canArchive = channel && channel.type !== 'dm' && channel.type !== 'group_dm';

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!channelId || !workspaceId) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500 dark:text-gray-400">
        Select a channel to start messaging
      </div>
    );
  }

  if (!channel) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500 dark:text-gray-400">
        Channel not found
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Channel header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-gray-500 dark:text-gray-400">
              {getChannelIcon(channel.type)}
            </span>
            <h1 className="font-semibold text-gray-900 dark:text-white">
              {channel.name}
            </h1>
          </div>

          {/* Settings menu */}
          {canArchive && (
            <div className="relative" ref={menuRef}>
              <button
                onClick={() => setIsMenuOpen(!isMenuOpen)}
                className="p-1.5 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                </svg>
              </button>

              {isMenuOpen && (
                <div className="absolute right-0 mt-1 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-10">
                  <button
                    onClick={() => {
                      setIsMenuOpen(false);
                      setIsArchiveModalOpen(true);
                    }}
                    className="w-full px-4 py-2 text-left text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                  >
                    Archive channel
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
        {channel.description && (
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {channel.description}
          </p>
        )}
      </div>

      {/* Archive confirmation modal */}
      <Modal
        isOpen={isArchiveModalOpen}
        onClose={() => setIsArchiveModalOpen(false)}
        title="Archive channel"
      >
        <p className="text-gray-600 dark:text-gray-300 mb-4">
          Are you sure you want to archive <strong>#{channel.name}</strong>? This channel will be hidden from the sidebar and members won't be able to send new messages.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setIsArchiveModalOpen(false)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={handleArchive}
            isLoading={archiveChannel.isPending}
          >
            Archive
          </Button>
        </div>
      </Modal>

      {/* Messages */}
      <MessageList
        channelId={channelId}
        lastReadMessageId={channel.last_read_message_id}
        unreadCount={channel.unread_count}
        onAtBottomChange={handleAtBottomChange}
      />

      {/* Composer */}
      <MessageComposer
        channelId={channelId}
        workspaceId={workspaceId}
        placeholder={`Message ${getChannelIcon(channel.type)}${channel.name}`}
      />
    </div>
  );
}
