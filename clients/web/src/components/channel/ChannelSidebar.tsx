import { useState, useMemo } from 'react';
import { Link, useParams, useNavigate } from 'react-router-dom';
import { useChannels, useWorkspace, useAuth } from '../../hooks';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { useUIStore } from '../../stores/uiStore';
import { ChannelListSkeleton, Modal, Button, Input, toast } from '../ui';
import { useCreateChannel, useMarkAllChannelsAsRead, useCreateDM } from '../../hooks/useChannels';
import { cn, getChannelIcon } from '../../lib/utils';
import type { ChannelWithMembership, ChannelType } from '@feather/api-client';

interface ChannelSidebarProps {
  workspaceId: string | undefined;
}

export function ChannelSidebar({ workspaceId }: ChannelSidebarProps) {
  const { channelId } = useParams<{ channelId: string }>();
  const { data: workspaceData } = useWorkspace(workspaceId);
  const { data, isLoading } = useChannels(workspaceId);
  const { toggleSidebar } = useUIStore();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isNewDMModalOpen, setIsNewDMModalOpen] = useState(false);
  const markAllAsRead = useMarkAllChannelsAsRead(workspaceId || '');

  const channels = data?.channels || [];
  const hasUnread = channels.some((c) => c.unread_count > 0);

  const groupedChannels = useMemo(() => {
    const groups = {
      public: [] as ChannelWithMembership[],
      private: [] as ChannelWithMembership[],
      dm: [] as ChannelWithMembership[],
    };

    channels.forEach((channel) => {
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
    <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-800">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h2 className="font-bold text-gray-900 dark:text-white truncate">
            {workspaceData?.workspace.name || 'Loading...'}
          </h2>
          <div className="flex items-center gap-1">
            {hasUnread && (
              <button
                onClick={() => markAllAsRead.mutate()}
                className="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
                title="Mark all as read"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </button>
            )}
            <button
              onClick={toggleSidebar}
              className="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 lg:hidden"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      {/* Channel List */}
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <ChannelListSkeleton />
        ) : (
          <>
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

            <ChannelSection
                title="Direct Messages"
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
          <CreateChannelModal
            isOpen={isCreateModalOpen}
            onClose={() => setIsCreateModalOpen(false)}
            workspaceId={workspaceId}
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
          <svg
            className={cn(
              'w-3 h-3 transition-transform',
              isExpanded ? 'rotate-90' : ''
            )}
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path d="M6 6L14 10L6 14V6Z" />
          </svg>
          <span>{title}</span>
        </button>
        {onAddClick && (
          <button
            onClick={onAddClick}
            className="p-0.5 hover:bg-gray-200 dark:hover:bg-gray-700 rounded hover:text-gray-700 dark:hover:text-gray-300"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
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

interface ChannelItemProps {
  channel: ChannelWithMembership;
  workspaceId: string;
  isActive: boolean;
}

function ChannelItem({ channel, workspaceId, isActive }: ChannelItemProps) {
  const icon = getChannelIcon(channel.type);
  const hasUnread = channel.unread_count > 0;
  const isDM = channel.type === 'dm' || channel.type === 'group_dm';
  const dmParticipant = isDM && channel.dm_participants?.[0];

  // For DMs, show participant name; for group DMs, show all names
  const displayName = isDM
    ? channel.dm_participants?.map((p) => p.display_name).join(', ') || channel.name
    : channel.name;

  return (
    <Link
      to={`/workspaces/${workspaceId}/channels/${channel.id}`}
      className={cn(
        'flex items-center gap-2 px-2 py-1.5 rounded text-sm',
        'hover:bg-gray-200 dark:hover:bg-gray-700',
        isActive
          ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
          : 'text-gray-700 dark:text-gray-300'
      )}
    >
      {isDM && dmParticipant ? (
        dmParticipant.avatar_url ? (
          <img
            src={dmParticipant.avatar_url}
            alt={dmParticipant.display_name}
            className="w-5 h-5 rounded-full"
          />
        ) : (
          <div className="w-5 h-5 rounded-full bg-gray-300 dark:bg-gray-600 flex items-center justify-center text-xs font-medium text-gray-600 dark:text-gray-300">
            {dmParticipant.display_name.charAt(0).toUpperCase()}
          </div>
        )
      ) : (
        icon && <span className="text-gray-500 dark:text-gray-400">{icon}</span>
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

function CreateChannelModal({
  isOpen,
  onClose,
  workspaceId,
}: {
  isOpen: boolean;
  onClose: () => void;
  workspaceId: string;
}) {
  const [name, setName] = useState('');
  const [type, setType] = useState<ChannelType>('public');
  const createChannel = useCreateChannel(workspaceId);

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    // Auto-format: lowercase, replace spaces with dashes, remove invalid chars
    const formatted = e.target.value
      .toLowerCase()
      .replace(/\s+/g, '-')
      .replace(/[^a-z0-9-]/g, '');
    setName(formatted);
  };

  const isValidName = name.length > 0 && CHANNEL_NAME_REGEX.test(name);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!isValidName) {
      toast('Channel name must contain only lowercase letters, numbers, and dashes', 'error');
      return;
    }

    try {
      await createChannel.mutateAsync({ name, type });
      toast('Channel created!', 'success');
      onClose();
      setName('');
      setType('public');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to create channel', 'error');
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Create Channel">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <Input
            label="Channel Name"
            value={name}
            onChange={handleNameChange}
            placeholder="general"
            required
          />
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Lowercase letters, numbers, and dashes only
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Channel Type
          </label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2">
              <input
                type="radio"
                value="public"
                checked={type === 'public'}
                onChange={() => setType('public')}
                className="text-primary-600"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">Public</span>
            </label>
            <label className="flex items-center gap-2">
              <input
                type="radio"
                value="private"
                checked={type === 'private'}
                onChange={() => setType('private')}
                className="text-primary-600"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">Private</span>
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" isLoading={createChannel.isPending}>
            Create
          </Button>
        </div>
      </form>
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

  const otherMembers = useMemo(() => {
    if (!membersData?.members || !user) return [];
    return membersData.members.filter((m) => m.user_id !== user.id);
  }, [membersData?.members, user]);

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
                  <div className="w-8 h-8 rounded-full bg-gray-300 dark:bg-gray-600 flex items-center justify-center text-sm font-medium text-gray-600 dark:text-gray-300">
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
          <Button type="button" variant="secondary" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={!selectedUserId} isLoading={createDM.isPending}>
            Start Conversation
          </Button>
        </div>
      </form>
    </Modal>
  );
}
