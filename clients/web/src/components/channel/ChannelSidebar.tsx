import { useState, useMemo } from 'react';
import { Link, useParams, useNavigate, useLocation } from 'react-router-dom';
import { CheckCircleIcon, ChevronRightIcon, PlusIcon, LockClosedIcon, HashtagIcon, InboxIcon } from '@heroicons/react/24/outline';
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  useDroppable,
  useDraggable,
} from '@dnd-kit/core';
import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core';
import { useChannels, useWorkspace, useAuth } from '../../hooks';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { ChannelListSkeleton, Modal, Button, Input, toast, Tabs, TabList, Tab, TabPanel, RadioGroup, Radio, Avatar } from '../ui';
import { useCreateChannel, useMarkAllChannelsAsRead, useCreateDM, useJoinChannel, useDMSuggestions, useStarChannel, useUnstarChannel } from '../../hooks/useChannels';
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
  const starChannel = useStarChannel(workspaceId || '');
  const unstarChannel = useUnstarChannel(workspaceId || '');
  const [activeChannel, setActiveChannel] = useState<ChannelWithMembership | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    })
  );

  // Local state for immediate visual updates during drag
  const [localChannels, setLocalChannels] = useState<ChannelWithMembership[]>([]);
  const [prevServerChannels, setPrevServerChannels] = useState<ChannelWithMembership[] | undefined>();

  // Sync local state with server state when server data changes (React-recommended pattern)
  if (data?.channels !== prevServerChannels) {
    setPrevServerChannels(data?.channels);
    if (data?.channels) {
      setLocalChannels(data.channels);
    }
  }

  const channels = localChannels;
  const totalUnreadCount = useMemo(() => channels.reduce((sum, c) => sum + c.unread_count, 0), [channels]);
  const hasUnread = totalUnreadCount > 0;
  const isUnreadsPage = location.pathname.includes('/unreads');

  const groupedChannels = useMemo(() => {
    const groups = {
      starred: [] as ChannelWithMembership[],
      channels: [] as ChannelWithMembership[],
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

      if (channel.type === 'public' || channel.type === 'private') {
        groups.channels.push(channel);
      } else {
        groups.dm.push(channel);
      }
    });

    // Sort starred: channels alphabetically first, then DMs alphabetically
    groups.starred.sort((a, b) => {
      const aIsDM = a.type === 'dm' || a.type === 'group_dm';
      const bIsDM = b.type === 'dm' || b.type === 'group_dm';
      if (aIsDM !== bIsDM) return aIsDM ? 1 : -1;
      return a.name.localeCompare(b.name);
    });

    // Sort channels alphabetically
    groups.channels.sort((a, b) => a.name.localeCompare(b.name));

    return groups;
  }, [channels]);

  // Determine if the active (dragged) item is a DM
  const isDraggingDM = activeChannel
    ? activeChannel.type === 'dm' || activeChannel.type === 'group_dm'
    : false;

  const handleDragStart = (event: DragStartEvent) => {
    const channel = channels.find((c) => c.id === event.active.id);
    if (channel) {
      setActiveChannel(channel);
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveChannel(null);

    if (!over) return;

    const channel = channels.find((c) => c.id === active.id);
    if (!channel) return;

    const overId = over.id as string;

    // Handle dropping on Starred section
    if (overId === 'starred-drop-zone' && !channel.is_starred) {
      // Update local state immediately for smooth visual transition
      setLocalChannels((prev) =>
        prev.map((c) => (c.id === channel.id ? { ...c, is_starred: true } : c))
      );
      starChannel.mutate(channel.id);
    }
    // Handle dropping on Channels or DMs sections (unstar)
    else if (
      (overId === 'channels-drop-zone' || overId === 'dms-drop-zone') &&
      channel.is_starred
    ) {
      // Update local state immediately for smooth visual transition
      setLocalChannels((prev) =>
        prev.map((c) => (c.id === channel.id ? { ...c, is_starred: false } : c))
      );
      unstarChannel.mutate(channel.id);
    }
  };

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
          <DndContext
            sensors={sensors}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
          >
            <DroppableChannelSection
              id="starred-drop-zone"
              title="Starred"
              channels={groupedChannels.starred}
              workspaceId={workspaceId}
              activeChannelId={channelId}
              showWhenEmpty={activeChannel !== null}
              isStarredSection
            />

            <DroppableChannelSection
              id="channels-drop-zone"
              title="Channels"
              channels={groupedChannels.channels}
              workspaceId={workspaceId}
              activeChannelId={channelId}
              onAddClick={() => setIsCreateModalOpen(true)}
              canDrop={activeChannel !== null && activeChannel.is_starred && !isDraggingDM}
            />

            <DroppableDMSection
              id="dms-drop-zone"
              channels={groupedChannels.dm}
              allChannels={channels}
              workspaceId={workspaceId}
              activeChannelId={channelId}
              onAddClick={() => setIsNewDMModalOpen(true)}
              canDrop={activeChannel !== null && activeChannel.is_starred && isDraggingDM}
            />

            <DragOverlay>
              {activeChannel && (
                <div className="bg-white dark:bg-gray-800 shadow-lg rounded px-2 py-1.5 text-sm text-gray-700 dark:text-gray-300 flex items-center gap-2">
                  <ChannelItemContent channel={activeChannel} />
                </div>
              )}
            </DragOverlay>
          </DndContext>
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

interface DroppableChannelSectionProps {
  id: string;
  title: string;
  channels: ChannelWithMembership[];
  workspaceId: string;
  activeChannelId: string | undefined;
  onAddClick?: () => void;
  showWhenEmpty?: boolean;
  isStarredSection?: boolean;
  canDrop?: boolean;
}

function DroppableChannelSection({
  id,
  title,
  channels,
  workspaceId,
  activeChannelId,
  onAddClick,
  showWhenEmpty = false,
  isStarredSection = false,
  canDrop = true,
}: DroppableChannelSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { isOver, setNodeRef } = useDroppable({ id });

  // Don't render section if empty and not configured to show when empty
  // Starred section always shows when expanded
  if (channels.length === 0 && !showWhenEmpty && !isStarredSection) {
    return null;
  }

  const showDropHighlight = isOver && canDrop;

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'py-2 transition-colors',
        showDropHighlight && 'bg-primary-50 dark:bg-primary-900/20'
      )}
    >
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
          {channels.length === 0 && showWhenEmpty && (
            <div className="px-2 py-3 text-xs text-gray-400 dark:text-gray-500 text-center border border-dashed border-gray-300 dark:border-gray-600 rounded">
              Drop here to star
            </div>
          )}
          {channels.length === 0 && !showWhenEmpty && isStarredSection && (
            <div className="px-2 py-1.5 text-xs text-gray-400 dark:text-gray-500">
              Drag important stuff here
            </div>
          )}
          {channels.map((channel) => (
            <DraggableChannelItem
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

interface DroppableDMSectionProps {
  id: string;
  channels: ChannelWithMembership[];
  allChannels: ChannelWithMembership[];
  workspaceId: string;
  activeChannelId: string | undefined;
  onAddClick?: () => void;
  canDrop?: boolean;
}

function DroppableDMSection({
  id,
  channels,
  allChannels,
  workspaceId,
  activeChannelId,
  onAddClick,
  canDrop = true,
}: DroppableDMSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const navigate = useNavigate();
  const { data: suggestionsData } = useDMSuggestions(workspaceId);
  const createDM = useCreateDM(workspaceId);
  const { isOver, setNodeRef } = useDroppable({ id });
  const showDropHighlight = isOver && canDrop;

  const handleSuggestedUserClick = async (userId: string) => {
    try {
      const result = await createDM.mutateAsync({ user_ids: [userId] });
      navigate(`/workspaces/${workspaceId}/channels/${result.channel.id}`);
    } catch {
      toast('Failed to start conversation', 'error');
    }
  };

  // Get user IDs that already have DM channels (including starred DMs)
  const existingDMUserIds = useMemo(() => {
    const ids = new Set<string>();
    allChannels.forEach((channel) => {
      if (channel.type === 'dm' && channel.dm_participants) {
        channel.dm_participants.forEach((p) => ids.add(p.user_id));
      }
    });
    return ids;
  }, [allChannels]);

  // Filter out suggested users who already have DMs
  const filteredSuggestions = useMemo(() => {
    if (!suggestionsData?.suggested_users) return [];
    return suggestionsData.suggested_users.filter(
      (user) => !existingDMUserIds.has(user.id)
    );
  }, [suggestionsData, existingDMUserIds]);

  // Show suggestions to fill up to ~5 total items
  const maxSuggestions = Math.max(0, 5 - channels.length);
  const suggestionsToShow = filteredSuggestions.slice(0, maxSuggestions);

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'py-2 transition-colors',
        showDropHighlight && 'bg-primary-50 dark:bg-primary-900/20'
      )}
    >
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
            <DraggableChannelItem
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

interface DraggableChannelItemProps {
  channel: ChannelWithMembership;
  workspaceId: string;
  isActive: boolean;
}

function DraggableChannelItem({ channel, workspaceId, isActive }: DraggableChannelItemProps) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: channel.id,
  });

  return (
    <div
      ref={setNodeRef}
      {...attributes}
      {...listeners}
      className={cn(isDragging && 'opacity-50')}
    >
      <ChannelItemLink channel={channel} workspaceId={workspaceId} isActive={isActive} />
    </div>
  );
}

interface ChannelItemLinkProps {
  channel: ChannelWithMembership;
  workspaceId: string;
  isActive: boolean;
}

function ChannelItemLink({ channel, workspaceId, isActive }: ChannelItemLinkProps) {
  const hasUnread = channel.unread_count > 0;

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
      <ChannelItemContent channel={channel} />
      {hasUnread && (
        <span className="ml-auto bg-primary-600 text-white text-xs px-1.5 py-0.5 rounded-full">
          {channel.unread_count}
        </span>
      )}
    </Link>
  );
}

interface ChannelItemContentProps {
  channel: ChannelWithMembership;
}

function ChannelItemContent({ channel }: ChannelItemContentProps) {
  const isDM = channel.type === 'dm' || channel.type === 'group_dm';
  const dmParticipant = isDM && channel.dm_participants?.[0];

  const rawPresence = useUserPresence(
    channel.type === 'dm' && dmParticipant ? dmParticipant.user_id : ''
  );
  const participantPresence = channel.type === 'dm' && dmParticipant
    ? (rawPresence ?? 'offline')
    : undefined;

  const displayName = isDM
    ? channel.dm_participants?.map((p) => p.display_name).join(', ') || channel.name
    : channel.name;

  const hasUnread = channel.unread_count > 0;

  return (
    <>
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
    </>
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
