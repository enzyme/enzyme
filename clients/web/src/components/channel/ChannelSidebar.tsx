import { useState, useMemo } from 'react';
import { Link, useParams, useNavigate, useLocation } from 'react-router-dom';
import { CheckCircleIcon, ChevronRightIcon, PlusIcon, LockClosedIcon, HashtagIcon, InboxIcon } from '@heroicons/react/24/outline';
import { useChannels, useWorkspace, useAuth } from '../../hooks';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { ChannelListSkeleton, Modal, Button, Input, toast, Tabs, TabList, Tab, TabPanel, RadioGroup, Radio, Avatar } from '../ui';
import { useCreateChannel, useMarkAllChannelsAsRead, useCreateDM, useJoinChannel, useDMSuggestions } from '../../hooks/useChannels';
import { cn, getAvatarColor } from '../../lib/utils';
import { useUserPresence } from '../../lib/presenceStore';
import type { ChannelWithMembership, ChannelType, SuggestedUser } from '@feather/api-client';

function ChannelIcon({ type, className }: { type: string; className?: string }) {
  if (type === 'private') {
    return <LockClosedIcon className={cn('w-4 h-4', className)} />;
  }
  if (type === 'public') {
    return <HashtagIcon className={cn('w-4 h-4', className)} />;
  }
  return null;
}

interface ChannelSidebarProps {
  workspaceId: string | undefined;
}

export function ChannelSidebar({ workspaceId }: ChannelSidebarProps) {
  const { channelId } = useParams<{ channelId: string }>();
  const location = useLocation();
  const { data: workspaceData } = useWorkspace(workspaceId);
  const { data, isLoading } = useChannels(workspaceId);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isNewDMModalOpen, setIsNewDMModalOpen] = useState(false);
  const markAllAsRead = useMarkAllChannelsAsRead(workspaceId || '');

  const channels = useMemo(() => data?.channels || [], [data?.channels]);
  const totalUnreadCount = useMemo(() => channels.reduce((sum, c) => sum + c.unread_count, 0), [channels]);
  const hasUnread = totalUnreadCount > 0;
  const isUnreadsPage = location.pathname.includes('/unreads');

  const groupedChannels = useMemo(() => {
    const groups = {
      starred: [] as ChannelWithMembership[],
      public: [] as ChannelWithMembership[],
      private: [] as ChannelWithMembership[],
      dm: [] as ChannelWithMembership[],
    };

    channels.forEach((channel) => {
      const isDM = channel.type === 'dm' || channel.type === 'group_dm';
      const isMember = channel.channel_role !== undefined;

      // DMs always show; other channels only if member
      if (!isDM && !isMember) return;

      // Starred channels go to starred section
      if (channel.is_starred) {
        groups.starred.push(channel);
        return;
      }

      if (channel.type === 'public') {
        groups.public.push(channel);
      } else if (channel.type === 'private') {
        groups.private.push(channel);
      } else {
        groups.dm.push(channel);
      }
    });

    return groups;
  }, [channels]);

  if (!workspaceId) {
    return null;
  }

  return (
    <div className="h-full flex flex-col bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h2 className="font-bold text-gray-900 dark:text-white truncate">
            {workspaceData?.workspace.name || 'Loading...'}
          </h2>
          {hasUnread && (
            <button
              onClick={() => markAllAsRead.mutate()}
              className="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              title="Mark all as read"
            >
              <CheckCircleIcon className="w-5 h-5" />
            </button>
          )}
        </div>
      </div>

      {/* Channel List */}
      <div className="flex-1 overflow-y-auto">
        {/* All Unreads link */}
        <div className="px-2 pt-2">
          <Link
            to={`/workspaces/${workspaceId}/unreads`}
            className={cn(
              'flex items-center gap-2 px-2 py-1.5 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-800/50',
              isUnreadsPage
                ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
                : 'text-gray-700 dark:text-gray-300'
            )}
          >
            <InboxIcon className="w-4 h-4 text-gray-500 dark:text-gray-400" />
            <span className={cn('truncate', hasUnread && 'font-semibold')}>All Unreads</span>
            {hasUnread && (
              <span className="ml-auto bg-primary-600 text-white text-xs px-1.5 py-0.5 rounded-full">
                {totalUnreadCount}
              </span>
            )}
          </Link>
        </div>

        {isLoading ? (
          <ChannelListSkeleton />
        ) : (
          <>
            {groupedChannels.starred.length > 0 && (
              <ChannelSection
                title="Starred"
                channels={groupedChannels.starred}
                workspaceId={workspaceId}
                activeChannelId={channelId}
              />
            )}

            <ChannelSection
              title="Channels"
              channels={groupedChannels.public}
              workspaceId={workspaceId}
              activeChannelId={channelId}
              onAddClick={() => setIsCreateModalOpen(true)}
            />

            {groupedChannels.private.length > 0 && (
              <ChannelSection
                title="Private Channels"
                channels={groupedChannels.private}
                workspaceId={workspaceId}
                activeChannelId={channelId}
              />
            )}

            <DMSection
                channels={groupedChannels.dm}
                workspaceId={workspaceId}
                activeChannelId={channelId}
                onAddClick={() => setIsNewDMModalOpen(true)}
              />
          </>
        )}
      </div>

      {workspaceId && (
        <>
          <ChannelBrowserModal
            isOpen={isCreateModalOpen}
            onClose={() => setIsCreateModalOpen(false)}
            workspaceId={workspaceId}
            channels={channels}
          />
          <NewDMModal
            isOpen={isNewDMModalOpen}
            onClose={() => setIsNewDMModalOpen(false)}
            workspaceId={workspaceId}
          />
        </>
      )}
    </div>
  );
}

interface ChannelSectionProps {
  title: string;
  channels: ChannelWithMembership[];
  workspaceId: string;
  activeChannelId: string | undefined;
  onAddClick?: () => void;
}

function ChannelSection({
  title,
  channels,
  workspaceId,
  activeChannelId,
  onAddClick,
}: ChannelSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  return (
    <div className="py-2">
      <div className="w-full flex items-center justify-between px-4 py-1 text-sm font-medium text-gray-500 dark:text-gray-400">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-1 hover:text-gray-700 dark:hover:text-gray-300"
        >
          <ChevronRightIcon
            className={cn('w-3 h-3 transition-transform', isExpanded && 'rotate-90')}
          />
          <span>{title}</span>
        </button>
        {onAddClick && (
          <button
            onClick={onAddClick}
            className="p-0.5 hover:bg-gray-200 dark:hover:bg-gray-700 rounded hover:text-gray-700 dark:hover:text-gray-300"
          >
            <PlusIcon className="w-4 h-4" />
          </button>
        )}
      </div>

      {isExpanded && (
        <div className="mt-1 space-y-0.5 px-2">
          {channels.map((channel) => (
            <ChannelItem
              key={channel.id}
              channel={channel}
              workspaceId={workspaceId}
              isActive={channel.id === activeChannelId}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface DMSectionProps {
  channels: ChannelWithMembership[];
  workspaceId: string;
  activeChannelId: string | undefined;
  onAddClick?: () => void;
}

function DMSection({
  channels,
  workspaceId,
  activeChannelId,
  onAddClick,
}: DMSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const navigate = useNavigate();
  const { data: suggestionsData } = useDMSuggestions(workspaceId);
  const createDM = useCreateDM(workspaceId);

  const handleSuggestedUserClick = async (userId: string) => {
    try {
      const result = await createDM.mutateAsync({ user_ids: [userId] });
      navigate(`/workspaces/${workspaceId}/channels/${result.channel.id}`);
    } catch {
      toast('Failed to start conversation', 'error');
    }
  };

  // Get user IDs that already have DM channels
  const existingDMUserIds = useMemo(() => {
    const ids = new Set<string>();
    channels.forEach((channel) => {
      if (channel.type === 'dm' && channel.dm_participants) {
        channel.dm_participants.forEach((p) => ids.add(p.user_id));
      }
    });
    return ids;
  }, [channels]);

  // Filter out suggested users who already have DMs
  const filteredSuggestions = useMemo(() => {
    if (!suggestionsData?.suggested_users) return [];
    return suggestionsData.suggested_users.filter(
      (user) => !existingDMUserIds.has(user.id)
    );
  }, [suggestionsData?.suggested_users, existingDMUserIds]);

  // Show suggestions to fill up to ~5 total items
  const maxSuggestions = Math.max(0, 5 - channels.length);
  const suggestionsToShow = filteredSuggestions.slice(0, maxSuggestions);

  return (
    <div className="py-2">
      <div className="w-full flex items-center justify-between px-4 py-1 text-sm font-medium text-gray-500 dark:text-gray-400">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-1 hover:text-gray-700 dark:hover:text-gray-300"
        >
          <ChevronRightIcon
            className={cn('w-3 h-3 transition-transform', isExpanded && 'rotate-90')}
          />
          <span>Direct Messages</span>
        </button>
        {onAddClick && (
          <button
            onClick={onAddClick}
            className="p-0.5 hover:bg-gray-200 dark:hover:bg-gray-700 rounded hover:text-gray-700 dark:hover:text-gray-300"
          >
            <PlusIcon className="w-4 h-4" />
          </button>
        )}
      </div>

      {isExpanded && (
        <div className="mt-1 space-y-0.5 px-2">
          {channels.map((channel) => (
            <ChannelItem
              key={channel.id}
              channel={channel}
              workspaceId={workspaceId}
              isActive={channel.id === activeChannelId}
            />
          ))}
          {suggestionsToShow.map((user) => (
            <SuggestedUserItem
              key={user.id}
              user={user}
              onClick={() => handleSuggestedUserClick(user.id)}
              isLoading={createDM.isPending}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface SuggestedUserItemProps {
  user: SuggestedUser;
  onClick: () => void;
  isLoading: boolean;
}

function SuggestedUserItem({ user, onClick, isLoading }: SuggestedUserItemProps) {
  const presence = useUserPresence(user.id);

  return (
    <button
      onClick={onClick}
      disabled={isLoading}
      className={cn(
        'w-full flex items-center gap-2 px-2 py-1.5 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-800/50 text-gray-500 dark:text-gray-400 text-left',
        isLoading && 'opacity-50 cursor-not-allowed'
      )}
    >
      <Avatar
        src={user.avatar_url}
        name={user.display_name}
        id={user.id}
        size="xs"
        status={presence ?? 'offline'}
      />
      <span className="truncate">{user.display_name}</span>
    </button>
  );
}

interface ChannelItemProps {
  channel: ChannelWithMembership;
  workspaceId: string;
  isActive: boolean;
}

function ChannelItem({ channel, workspaceId, isActive }: ChannelItemProps) {
  const hasUnread = channel.unread_count > 0;
  const isDM = channel.type === 'dm' || channel.type === 'group_dm';
  const dmParticipant = isDM && channel.dm_participants?.[0];

  // Get presence for 1:1 DMs only (group DMs have multiple participants)
  const rawPresence = useUserPresence(
    channel.type === 'dm' && dmParticipant ? dmParticipant.user_id : ''
  );
  // Default to offline if not in presence store (only online users are tracked)
  const participantPresence = channel.type === 'dm' && dmParticipant
    ? (rawPresence ?? 'offline')
    : undefined;

  // For DMs, show participant name; for group DMs, show all names
  const displayName = isDM
    ? channel.dm_participants?.map((p) => p.display_name).join(', ') || channel.name
    : channel.name;

  return (
    <Link
      to={`/workspaces/${workspaceId}/channels/${channel.id}`}
      className={cn(
        'flex items-center gap-2 px-2 py-1.5 rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-800/50',
        isActive
          ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
          : 'text-gray-700 dark:text-gray-300'
      )}
    >
      {isDM && dmParticipant ? (
        <Avatar
          src={dmParticipant.avatar_url}
          name={dmParticipant.display_name}
          id={dmParticipant.user_id}
          size="xs"
          status={participantPresence}
        />
      ) : (
        <ChannelIcon type={channel.type} className="text-gray-500 dark:text-gray-400" />
      )}
      <span className={cn('truncate', hasUnread && 'font-semibold')}>{displayName}</span>
      {hasUnread && (
        <span className="ml-auto bg-primary-600 text-white text-xs px-1.5 py-0.5 rounded-full">
          {channel.unread_count}
        </span>
      )}
    </Link>
  );
}

const CHANNEL_NAME_REGEX = /^[a-z0-9]+(-[a-z0-9]+)*$/;

function ChannelBrowserModal({
  isOpen,
  onClose,
  workspaceId,
  channels,
}: {
  isOpen: boolean;
  onClose: () => void;
  workspaceId: string;
  channels: ChannelWithMembership[];
}) {
  const navigate = useNavigate();
  const [tab, setTab] = useState<'browse' | 'create'>('browse');
  const [name, setName] = useState('');
  const [type, setType] = useState<ChannelType>('public');
  const createChannel = useCreateChannel(workspaceId);
  const joinChannel = useJoinChannel(workspaceId);
  const [joiningId, setJoiningId] = useState<string | null>(null);

  // Public channels the user hasn't joined
  const unjoinedChannels = useMemo(() => {
    return channels.filter(
      (c) => c.type === 'public' && c.channel_role === undefined
    );
  }, [channels]);

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const formatted = e.target.value
      .toLowerCase()
      .replace(/\s+/g, '-')
      .replace(/[^a-z0-9-]/g, '');
    setName(formatted);
  };

  const isValidName = name.length > 0 && CHANNEL_NAME_REGEX.test(name);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!isValidName) {
      toast('Channel name must contain only lowercase letters, numbers, and dashes', 'error');
      return;
    }

    try {
      const result = await createChannel.mutateAsync({ name, type });
      toast('Channel created!', 'success');
      onClose();
      setName('');
      setType('public');
      setTab('browse');
      navigate(`/workspaces/${workspaceId}/channels/${result.channel.id}`);
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to create channel', 'error');
    }
  };

  const handleJoin = async (channelId: string) => {
    setJoiningId(channelId);
    try {
      await joinChannel.mutateAsync(channelId);
      toast('Joined channel', 'success');
      onClose();
      navigate(`/workspaces/${workspaceId}/channels/${channelId}`);
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to join channel', 'error');
    } finally {
      setJoiningId(null);
    }
  };

  const handleClose = () => {
    onClose();
    setName('');
    setType('public');
    setTab('browse');
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Channels">
      <Tabs selectedKey={tab} onSelectionChange={(key) => setTab(key as 'browse' | 'create')} className="mb-4">
        <TabList>
          <Tab id="browse">Browse</Tab>
          <Tab id="create">Create New</Tab>
        </TabList>
        <TabPanel id="browse" className="mt-4">
          <div className="space-y-1 max-h-64 overflow-y-auto">
            {unjoinedChannels.length === 0 ? (
              <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
                No channels to join. Create a new one!
              </p>
            ) : (
              unjoinedChannels.map((channel) => (
                <div
                  key={channel.id}
                  className="flex items-center justify-between px-3 py-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                >
                  <div className="flex items-center gap-2">
                    <span className="text-gray-500 dark:text-gray-400">#</span>
                    <span className="text-gray-900 dark:text-white">{channel.name}</span>
                  </div>
                  <Button
                    size="sm"
                    variant="secondary"
                    onPress={() => handleJoin(channel.id)}
                    isLoading={joiningId === channel.id}
                    isDisabled={joiningId !== null}
                  >
                    Join
                  </Button>
                </div>
              ))
            )}
          </div>
        </TabPanel>
        <TabPanel id="create" className="mt-4">
          <form onSubmit={handleCreate} className="space-y-4">
            <div>
              <Input
                label="Channel Name"
                value={name}
                onChange={handleNameChange}
                placeholder="general"
                isRequired
              />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Lowercase letters, numbers, and dashes only
              </p>
            </div>

            <RadioGroup
              label="Channel Type"
              value={type}
              onChange={(value) => setType(value as ChannelType)}
            >
              <Radio value="public">Public</Radio>
              <Radio value="private">Private</Radio>
            </RadioGroup>

            <div className="flex justify-end gap-2">
              <Button type="button" variant="secondary" onPress={handleClose}>
                Cancel
              </Button>
              <Button type="submit" isLoading={createChannel.isPending}>
                Create
              </Button>
            </div>
          </form>
        </TabPanel>
      </Tabs>
    </Modal>
  );
}

function NewDMModal({
  isOpen,
  onClose,
  workspaceId,
}: {
  isOpen: boolean;
  onClose: () => void;
  workspaceId: string;
}) {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const createDM = useCreateDM(workspaceId);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);

  const members = membersData?.members;
  const otherMembers = useMemo(() => {
    if (!members || !user) return [];
    return members.filter((m) => m.user_id !== user.id);
  }, [members, user]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUserId) return;

    try {
      const result = await createDM.mutateAsync({ user_ids: [selectedUserId] });
      onClose();
      setSelectedUserId(null);
      navigate(`/workspaces/${workspaceId}/channels/${result.channel.id}`);
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to start conversation', 'error');
    }
  };

  const handleClose = () => {
    onClose();
    setSelectedUserId(null);
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="New Message">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="max-h-64 overflow-y-auto space-y-1">
          {otherMembers.map((member) => {
            const displayName = member.display_name_override || member.display_name;
            const isSelected = selectedUserId === member.user_id;

            return (
              <button
                key={member.user_id}
                type="button"
                onClick={() => setSelectedUserId(member.user_id)}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-2 rounded text-left',
                  isSelected
                    ? 'bg-primary-100 dark:bg-primary-900/30'
                    : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                )}
              >
                {member.avatar_url ? (
                  <img
                    src={member.avatar_url}
                    alt={displayName}
                    className="w-8 h-8 rounded-full"
                  />
                ) : (
                  <div className={cn('w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium text-white', getAvatarColor(member.user_id))}>
                    {displayName.charAt(0).toUpperCase()}
                  </div>
                )}
                <span className="text-gray-900 dark:text-white">{displayName}</span>
              </button>
            );
          })}
          {otherMembers.length === 0 && (
            <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
              No other members in this workspace
            </p>
          )}
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onPress={handleClose}>
            Cancel
          </Button>
          <Button type="submit" isDisabled={!selectedUserId} isLoading={createDM.isPending}>
            Start Conversation
          </Button>
        </div>
      </form>
    </Modal>
  );
}
