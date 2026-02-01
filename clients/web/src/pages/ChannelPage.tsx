import { useState, useRef, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { EllipsisVerticalIcon, LockClosedIcon, HashtagIcon } from '@heroicons/react/24/outline';
import { useChannels, useArchiveChannel, useLeaveChannel, useJoinChannel, useAuth } from '../hooks';
import { useThreadPanel } from '../hooks/usePanel';
import { useMarkChannelAsRead } from '../hooks/useChannels';
import { MessageList, MessageComposer } from '../components/message';
import { ChannelMembersButton } from '../components/channel/ChannelMembersButton';
import { Spinner, Modal, Button, toast } from '../components/ui';

function ChannelIcon({ type, className }: { type: string; className?: string }) {
  if (type === 'private') {
    return <LockClosedIcon className={className} />;
  }
  if (type === 'public') {
    return <HashtagIcon className={className} />;
  }
  return null;
}

function getChannelPrefix(type: string): string {
  return type === 'private' ? '' : '#';
}

export function ChannelPage() {
  const { workspaceId, channelId } = useParams<{
    workspaceId: string;
    channelId: string;
  }>();
  const navigate = useNavigate();

  const queryClient = useQueryClient();
  const { workspaces } = useAuth();
  const { data: channelsData, isLoading } = useChannels(workspaceId);
  const { closeThread } = useThreadPanel();

  const channel = channelsData?.channels.find((c) => c.id === channelId);
  const archiveChannel = useArchiveChannel(workspaceId || '');
  const leaveChannel = useLeaveChannel(workspaceId || '');
  const joinChannel = useJoinChannel(workspaceId || '');
  const markAsRead = useMarkChannelAsRead(workspaceId || '');

  // Get user's role in this workspace
  const workspaceMembership = workspaces?.find((w) => w.id === workspaceId);
  const canAddMembers =
    workspaceMembership?.role === 'admin' ||
    workspaceMembership?.role === 'owner' ||
    channel?.channel_role !== undefined; // User is channel member

  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isArchiveModalOpen, setIsArchiveModalOpen] = useState(false);
  const [isLeaveModalOpen, setIsLeaveModalOpen] = useState(false);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const menuRef = useRef<HTMLDivElement>(null);
  const markAsReadTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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
  // Note: closeThread intentionally omitted from deps - we only want to run
  // this when channelId changes, not when the callback reference changes
  useEffect(() => {
    setIsAtBottom(true);
    closeThread();
    // eslint-disable-next-line react-hooks/exhaustive-deps
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

  const handleLeave = async () => {
    if (!channelId || !workspaceId) return;
    try {
      await leaveChannel.mutateAsync(channelId);
      toast('Left channel', 'success');
      setIsLeaveModalOpen(false);
      // Navigate to another joined channel, or workspace root if none
      const otherChannel = channelsData?.channels.find(
        (c) => c.id !== channelId && c.channel_role !== undefined
      );
      if (otherChannel) {
        navigate(`/workspaces/${workspaceId}/channels/${otherChannel.id}`);
      } else {
        navigate(`/workspaces/${workspaceId}`);
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to leave channel', 'error');
    }
  };

  const handleJoin = async () => {
    if (!channelId || !workspaceId) return;
    try {
      await joinChannel.mutateAsync(channelId);
      // Wait for the channels query to refetch so the UI updates
      await queryClient.refetchQueries({ queryKey: ['channels', workspaceId] });
      toast('Joined channel', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to join channel', 'error');
    }
  };

  const isMember = channel?.channel_role !== undefined;
  const canJoin = channel && channel.type === 'public' && !isMember;
  const canArchive = channel && channel.type !== 'dm' && channel.type !== 'group_dm';
  const canLeave = channel && channel.type !== 'dm' && channel.type !== 'group_dm' && isMember;

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
            <ChannelIcon type={channel.type} className="w-5 h-5 text-gray-500 dark:text-gray-400" />
            <h1 className="font-semibold text-gray-900 dark:text-white">
              {channel.name}
            </h1>
          </div>

          <div className="flex items-center gap-1">
            {/* Channel members */}
            <ChannelMembersButton
              channelId={channelId}
              workspaceId={workspaceId}
              channelType={channel.type}
              canAddMembers={canAddMembers}
            />

            {/* Settings menu */}
            {(canArchive || canLeave) && (
              <div className="relative" ref={menuRef}>
                <button
                  onClick={() => setIsMenuOpen(!isMenuOpen)}
                  className="p-1.5 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
                >
                  <EllipsisVerticalIcon className="w-5 h-5" />
                </button>

                {isMenuOpen && (
                  <div className="absolute right-0 mt-1 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-10">
                    {canLeave && (
                      <button
                        onClick={() => {
                          setIsMenuOpen(false);
                          setIsLeaveModalOpen(true);
                        }}
                        className="w-full px-3 py-1.5 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                      >
                        Leave channel
                      </button>
                    )}
                    {canArchive && (
                      <button
                        onClick={() => {
                          setIsMenuOpen(false);
                          setIsArchiveModalOpen(true);
                        }}
                        className="w-full px-3 py-1.5 text-left text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                      >
                        Archive channel
                      </button>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
        {channel.description && (
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {channel.description}
          </p>
        )}
      </div>

      {/* Leave confirmation modal */}
      <Modal
        isOpen={isLeaveModalOpen}
        onClose={() => setIsLeaveModalOpen(false)}
        title="Leave channel"
      >
        <p className="text-gray-600 dark:text-gray-300 mb-4">
          Are you sure you want to leave <strong>#{channel.name}</strong>? You can rejoin anytime if it's a public channel.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setIsLeaveModalOpen(false)}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleLeave}
            isLoading={leaveChannel.isPending}
          >
            Leave
          </Button>
        </div>
      </Modal>

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

      {/* Messages - always visible */}
      <MessageList
        channelId={channelId}
        lastReadMessageId={channel.last_read_message_id}
        unreadCount={channel.unread_count}
        onAtBottomChange={handleAtBottomChange}
      />

      {canJoin ? (
        /* Join banner for non-member channels */
        <div className="flex-shrink-0 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-gray-600 dark:text-gray-300">
              <span className="text-gray-400">#</span>
              <span>You're viewing <strong>{channel.name}</strong></span>
            </div>
            <Button
              onClick={handleJoin}
              isLoading={joinChannel.isPending}
              size="sm"
            >
              Join Channel
            </Button>
          </div>
        </div>
      ) : (
        /* Composer for members */
        <MessageComposer
          channelId={channelId}
          workspaceId={workspaceId}
          placeholder={`Message ${getChannelPrefix(channel.type)}${channel.name}`}
        />
      )}
    </div>
  );
}
