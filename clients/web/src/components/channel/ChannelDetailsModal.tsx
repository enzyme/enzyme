import { useState, useEffect } from 'react';
import { Modal, Avatar, Button, Tabs, TabList, Tab, TabPanel, Spinner } from '../ui';
import { useChannelMembers, useAddChannelMember, useUpdateChannel } from '../../hooks/useChannels';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { cn } from '../../lib/utils';
import type {
  ChannelMember,
  WorkspaceMemberWithUser,
  ChannelWithMembership,
} from '@feather/api-client';

type TabId = 'about' | 'members' | 'add';

interface ChannelDetailsModalProps {
  isOpen: boolean;
  onClose: () => void;
  channelId: string;
  workspaceId: string;
  canAddMembers: boolean;
  canEditChannel: boolean;
  channel: ChannelWithMembership;
  defaultTab?: TabId;
}

export function ChannelDetailsModal({
  isOpen,
  onClose,
  channelId,
  workspaceId,
  canAddMembers,
  canEditChannel,
  channel,
  defaultTab = 'about',
}: ChannelDetailsModalProps) {
  const { data: membersData, isLoading: membersLoading } = useChannelMembers(channelId);
  const { data: workspaceMembersData, isLoading: workspaceMembersLoading } =
    useWorkspaceMembers(workspaceId);
  const addMember = useAddChannelMember(channelId);
  const updateChannel = useUpdateChannel(workspaceId, channelId);
  const [addingUserId, setAddingUserId] = useState<string | null>(null);
  const [description, setDescription] = useState(channel.description || '');
  const [selectedTab, setSelectedTab] = useState<TabId>(defaultTab);

  // Reset description when modal opens or channel changes
  useEffect(() => {
    if (isOpen) {
      setDescription(channel.description || '');
      setSelectedTab(defaultTab);
    }
  }, [isOpen, channel.description, defaultTab]);

  const members = membersData?.members || [];
  const workspaceMembers = workspaceMembersData?.members || [];

  // Filter workspace members to only show those not already in the channel
  const channelMemberIds = new Set(members.map((m) => m.user_id));
  const nonMembers = workspaceMembers.filter((m) => !channelMemberIds.has(m.user_id));

  const handleAddMember = async (userId: string) => {
    setAddingUserId(userId);
    try {
      await addMember.mutateAsync({ userId });
    } finally {
      setAddingUserId(null);
    }
  };

  const handleSaveDescription = async () => {
    await updateChannel.mutateAsync({ description });
  };

  const hasDescriptionChanged = description !== (channel.description || '');

  const getRoleBadge = (role?: string) => {
    if (!role || role === 'poster') return null;
    return (
      <span
        className={cn(
          'rounded px-1.5 py-0.5 text-xs',
          role === 'admin'
            ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
            : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400',
        )}
      >
        {role}
      </span>
    );
  };

  const renderAboutTab = () => (
    <div className="space-y-4">
      <div>
        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Channel name
        </label>
        <p className="text-gray-900 dark:text-white">{channel.name}</p>
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Description
        </label>
        {canEditChannel ? (
          <>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Add a description for this channel..."
              className="w-full resize-none rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-transparent focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
              rows={3}
            />
            <div className="mt-2 flex justify-end">
              <Button
                size="sm"
                onPress={handleSaveDescription}
                isLoading={updateChannel.isPending}
                isDisabled={!hasDescriptionChanged}
              >
                Save
              </Button>
            </div>
          </>
        ) : (
          <p className="text-gray-600 dark:text-gray-400">
            {channel.description || 'No description set'}
          </p>
        )}
      </div>
    </div>
  );

  const renderMemberList = (membersList: ChannelMember[]) => (
    <div className="max-h-64 space-y-1 overflow-y-auto">
      {membersList.map((member) => (
        <div
          key={member.user_id}
          className="flex items-center gap-3 rounded px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/50"
        >
          <Avatar
            src={member.avatar_url}
            name={member.display_name}
            id={member.user_id}
            size="sm"
          />
          <span className="flex-1 text-gray-900 dark:text-white">{member.display_name}</span>
          {getRoleBadge(member.channel_role)}
        </div>
      ))}
      {membersList.length === 0 && (
        <p className="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
          No members found
        </p>
      )}
    </div>
  );

  const renderAddMemberList = (membersList: WorkspaceMemberWithUser[]) => (
    <div className="max-h-64 space-y-1 overflow-y-auto">
      {membersList.map((member) => {
        const displayName = member.display_name_override || member.display_name;
        const isAdding = addingUserId === member.user_id;

        return (
          <div
            key={member.user_id}
            className="flex items-center gap-3 rounded px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/50"
          >
            <Avatar src={member.avatar_url} name={displayName} id={member.user_id} size="sm" />
            <span className="flex-1 text-gray-900 dark:text-white">{displayName}</span>
            <Button
              size="sm"
              variant="secondary"
              onPress={() => handleAddMember(member.user_id)}
              isLoading={isAdding}
              isDisabled={addMember.isPending}
            >
              Add
            </Button>
          </div>
        );
      })}
      {membersList.length === 0 && (
        <p className="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
          All workspace members are already in this channel
        </p>
      )}
    </div>
  );

  const isGroupDM = channel.type === 'group_dm';
  const isLoading = membersLoading || (canAddMembers && workspaceMembersLoading);

  // For group DMs, default to members tab since there's no about tab
  const effectiveTab = isGroupDM && selectedTab === 'about' ? 'members' : selectedTab;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isGroupDM ? 'Conversation Details' : 'Channel Details'}
      size="md"
    >
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Spinner />
        </div>
      ) : (
        <Tabs selectedKey={effectiveTab} onSelectionChange={(key) => setSelectedTab(key as TabId)}>
          <TabList>
            {!isGroupDM && <Tab id="about">About</Tab>}
            <Tab id="members">Members ({members.length})</Tab>
            {canAddMembers && <Tab id="add">Add Members</Tab>}
          </TabList>
          {!isGroupDM && (
            <TabPanel id="about" className="pt-4">
              {renderAboutTab()}
            </TabPanel>
          )}
          <TabPanel id="members" className="pt-4">
            {renderMemberList(members)}
          </TabPanel>
          {canAddMembers && (
            <TabPanel id="add" className="pt-4">
              {renderAddMemberList(nonMembers)}
            </TabPanel>
          )}
        </Tabs>
      )}
    </Modal>
  );
}
