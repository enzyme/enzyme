import { useState, useRef, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import {
  EllipsisVerticalIcon,
  LockClosedIcon,
  HashtagIcon,
  StarIcon,
  ArrowLeftStartOnRectangleIcon,
  ArchiveBoxIcon,
} from '@heroicons/react/24/outline';
import { StarIcon as StarIconSolid } from '@heroicons/react/24/solid';
import { Button as AriaButton } from 'react-aria-components';
import {
  useChannels,
  useArchiveChannel,
  useLeaveChannel,
  useJoinChannel,
  useAuth,
  useAutoFocusComposer,
} from '../hooks';
import { useThreadPanel } from '../hooks/usePanel';
import { useMarkChannelAsRead, useStarChannel, useUnstarChannel } from '../hooks/useChannels';
import { MessageList, MessageComposer, type MessageComposerRef } from '../components/message';
import { ChannelMembersButton } from '../components/channel/ChannelMembersButton';
import { ChannelNotificationButton } from '../components/channel/ChannelNotificationButton';
import { ChannelDetailsModal } from '../components/channel/ChannelDetailsModal';
import { ConvertToChannelModal } from '../components/channel/ConvertToChannelModal';
import { Spinner, Modal, Button, toast, Tooltip } from '../components/ui';

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
  const { threadId, closeThread } = useThreadPanel();
  const composerRef = useRef<MessageComposerRef>(null);

  const channel = channelsData?.channels.find((c) => c.id === channelId);
  const archiveChannel = useArchiveChannel(workspaceId || '');
  const leaveChannel = useLeaveChannel(workspaceId || '');
  const joinChannel = useJoinChannel(workspaceId || '');
  const markAsRead = useMarkChannelAsRead(workspaceId || '');
  const starChannel = useStarChannel(workspaceId || '');
  const unstarChannel = useUnstarChannel(workspaceId || '');

  // Get user's role in this workspace
  const workspaceMembership = workspaces?.find((w) => w.id === workspaceId);
  const canAddMembers =
    workspaceMembership?.role === 'admin' ||
    workspaceMembership?.role === 'owner' ||
    channel?.channel_role !== undefined; // User is channel member

  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isArchiveModalOpen, setIsArchiveModalOpen] = useState(false);
  const [isLeaveModalOpen, setIsLeaveModalOpen] = useState(false);
  const [isConvertModalOpen, setIsConvertModalOpen] = useState(false);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const [isDetailsModalOpen, setIsDetailsModalOpen] = useState(false);
  const [detailsModalTab, setDetailsModalTab] = useState<'about' | 'members' | 'add'>('about');
  const menuRef = useRef<HTMLDivElement>(null);
  const markAsReadTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const descriptionRef = useRef<HTMLElement>(null);
  const [isDescriptionTruncated, setIsDescriptionTruncated] = useState(false);

  // Check if description is truncated
  useEffect(() => {
    const checkTruncation = () => {
      const el = descriptionRef.current;
      if (el) {
        setIsDescriptionTruncated(el.scrollWidth > el.clientWidth);
      } else {
        setIsDescriptionTruncated(false);
      }
    };

    checkTruncation();
    window.addEventListener('resize', checkTruncation);
    return () => window.removeEventListener('resize', checkTruncation);
  }, [channel?.description]);

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

  // Reset state when changing channels (but not on initial mount, to preserve deep links)
  const prevChannelIdRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    setIsAtBottom(true);
    const prevChannelId = prevChannelIdRef.current;
    prevChannelIdRef.current = channelId;

    if (prevChannelId && prevChannelId !== channelId) {
      closeThread();
    }
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
        (c) => c.id !== channelId && c.channel_role !== undefined,
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
  const canArchive =
    channel && channel.type !== 'dm' && channel.type !== 'group_dm' && !channel.is_default;
  const canLeave = channel && channel.type !== 'dm' && !channel.is_default && isMember;
  const canConvert = channel && channel.type === 'group_dm' && isMember;
  const canEditChannel =
    channel &&
    channel.type !== 'dm' &&
    channel.type !== 'group_dm' &&
    channel.channel_role !== undefined;

  // Auto-focus main composer only when no thread is open and user is a member
  useAutoFocusComposer(composerRef, isMember && !threadId);

  const openDetailsModal = (tab: 'about' | 'members' | 'add') => {
    setDetailsModalTab(tab);
    setIsDetailsModalOpen(true);
  };

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!channelId || !workspaceId) {
    return (
      <div className="flex flex-1 items-center justify-center text-gray-500 dark:text-gray-400">
        Select a channel to start messaging
      </div>
    );
  }

  if (!channel) {
    return (
      <div className="flex flex-1 items-center justify-center text-gray-500 dark:text-gray-400">
        Channel not found
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Channel header */}
      <div className="flex-shrink-0 border-b border-gray-200 bg-white px-4 py-3 dark:border-gray-700 dark:bg-gray-900">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2">
            {isMember && (
              <button
                onClick={() => {
                  if (channel.is_starred) {
                    unstarChannel.mutate(channelId);
                  } else {
                    starChannel.mutate(channelId);
                  }
                }}
                className="flex-shrink-0 p-1 text-gray-400 transition-colors hover:text-yellow-500 dark:hover:text-yellow-400"
                title={channel.is_starred ? 'Unstar channel' : 'Star channel'}
              >
                {channel.is_starred ? (
                  <StarIconSolid className="h-5 w-5 text-yellow-500" />
                ) : (
                  <StarIcon className="h-5 w-5" />
                )}
              </button>
            )}
            <button
              onClick={() => openDetailsModal('about')}
              className="-ml-1.5 flex flex-shrink-0 items-center gap-2 rounded px-1.5 py-0.5 transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
            >
              <ChannelIcon
                type={channel.type}
                className="h-5 w-5 text-gray-500 dark:text-gray-400"
              />
              <h1 className="font-semibold text-gray-900 dark:text-white">{channel.name}</h1>
            </button>
            {channel.description &&
              (isDescriptionTruncated ? (
                <Tooltip content={channel.description} placement="bottom">
                  <AriaButton
                    ref={descriptionRef as React.RefObject<HTMLButtonElement>}
                    className="min-w-0 max-w-md cursor-default truncate border-none bg-transparent p-0 text-left text-sm text-gray-400 outline-none dark:text-gray-500"
                    excludeFromTabOrder
                  >
                    {channel.description}
                  </AriaButton>
                </Tooltip>
              ) : (
                <span
                  ref={descriptionRef as React.RefObject<HTMLSpanElement>}
                  className="min-w-0 max-w-md truncate text-sm text-gray-400 dark:text-gray-500"
                >
                  {channel.description}
                </span>
              ))}
          </div>

          <div className="flex flex-shrink-0 items-center gap-1">
            {/* Channel members */}
            <ChannelMembersButton
              channelId={channelId}
              channelType={channel.type}
              onPress={() => openDetailsModal('members')}
            />

            {/* Notification settings */}
            <ChannelNotificationButton channelId={channelId} channelType={channel.type} />

            {/* Settings menu */}
            {(canArchive || canLeave || canConvert) && (
              <div className="relative" ref={menuRef}>
                <button
                  onClick={() => setIsMenuOpen(!isMenuOpen)}
                  className="rounded p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-gray-800 dark:hover:text-gray-300"
                >
                  <EllipsisVerticalIcon className="h-5 w-5" />
                </button>

                {isMenuOpen && (
                  <div className="absolute right-0 z-10 mt-1 w-48 rounded-md border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-800">
                    {canConvert && (
                      <button
                        onClick={() => {
                          setIsMenuOpen(false);
                          setIsConvertModalOpen(true);
                        }}
                        className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                      >
                        <HashtagIcon className="h-4 w-4" />
                        Convert to channel
                      </button>
                    )}
                    {canLeave && (
                      <button
                        onClick={() => {
                          setIsMenuOpen(false);
                          setIsLeaveModalOpen(true);
                        }}
                        className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                      >
                        <ArrowLeftStartOnRectangleIcon className="h-4 w-4" />
                        {channel.type === 'group_dm' ? 'Leave direct message' : 'Leave channel'}
                      </button>
                    )}
                    {canArchive && (
                      <>
                        <div className="my-1 border-t border-gray-200 dark:border-gray-700" />
                        <button
                          onClick={() => {
                            setIsMenuOpen(false);
                            setIsArchiveModalOpen(true);
                          }}
                          className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-red-600 hover:bg-gray-100 dark:text-red-400 dark:hover:bg-gray-700"
                        >
                          <ArchiveBoxIcon className="h-4 w-4" />
                          Archive channel
                        </button>
                      </>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Leave confirmation modal */}
      <Modal
        isOpen={isLeaveModalOpen}
        onClose={() => setIsLeaveModalOpen(false)}
        title={channel.type === 'group_dm' ? 'Leave conversation' : 'Leave channel'}
      >
        <p className="mb-4 text-gray-600 dark:text-gray-300">
          {channel.type === 'group_dm' ? (
            <>
              Are you sure you want to leave this group conversation? You will lose access to the
              message history.
            </>
          ) : (
            <>
              Are you sure you want to leave <strong>#{channel.name}</strong>? You can rejoin
              anytime if it's a public channel.
            </>
          )}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setIsLeaveModalOpen(false)}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleLeave} isLoading={leaveChannel.isPending}>
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
        <p className="mb-4 text-gray-600 dark:text-gray-300">
          Are you sure you want to archive <strong>#{channel.name}</strong>? This channel will be
          hidden from the sidebar and members won't be able to send new messages.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setIsArchiveModalOpen(false)}>
            Cancel
          </Button>
          <Button variant="danger" onClick={handleArchive} isLoading={archiveChannel.isPending}>
            Archive
          </Button>
        </div>
      </Modal>

      {/* Convert to channel modal */}
      <ConvertToChannelModal
        isOpen={isConvertModalOpen}
        onClose={() => setIsConvertModalOpen(false)}
        workspaceId={workspaceId}
        channelId={channelId}
      />

      {/* Messages - always visible */}
      <MessageList
        channelId={channelId}
        lastReadMessageId={channel.last_read_message_id}
        unreadCount={channel.unread_count}
        onAtBottomChange={handleAtBottomChange}
      />

      {canJoin ? (
        /* Join banner for non-member channels */
        <div className="flex-shrink-0 border-t border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-700 dark:bg-gray-800">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-gray-600 dark:text-gray-300">
              <span className="text-gray-400">#</span>
              <span>
                You're viewing <strong>{channel.name}</strong>
              </span>
            </div>
            <Button onClick={handleJoin} isLoading={joinChannel.isPending} size="sm">
              Join Channel
            </Button>
          </div>
        </div>
      ) : (
        /* Composer for members */
        <MessageComposer
          ref={composerRef}
          channelId={channelId}
          workspaceId={workspaceId}
          placeholder={`Message ${getChannelPrefix(channel.type)}${channel.name}`}
        />
      )}

      {/* Channel details modal */}
      <ChannelDetailsModal
        isOpen={isDetailsModalOpen}
        onClose={() => setIsDetailsModalOpen(false)}
        channelId={channelId}
        workspaceId={workspaceId}
        channel={channel}
        canAddMembers={canAddMembers}
        canEditChannel={!!canEditChannel}
        defaultTab={detailsModalTab}
      />
    </div>
  );
}
