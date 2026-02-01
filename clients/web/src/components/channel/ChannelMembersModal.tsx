import { useState } from 'react';
import { Modal, Avatar, Button, Tabs, TabList, Tab, TabPanel, Spinner } from '../ui';
import { useChannelMembers, useAddChannelMember } from '../../hooks/useChannels';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { cn } from '../../lib/utils';
import type { ChannelMember, WorkspaceMemberWithUser } from '@feather/api-client';

interface ChannelMembersModalProps {
  isOpen: boolean;
  onClose: () => void;
  channelId: string;
  workspaceId: string;
  canAddMembers: boolean;
}

export function ChannelMembersModal({
  isOpen,
  onClose,
  channelId,
  workspaceId,
  canAddMembers,
}: ChannelMembersModalProps) {
  const { data: membersData, isLoading: membersLoading } = useChannelMembers(channelId);
  const { data: workspaceMembersData, isLoading: workspaceMembersLoading } = useWorkspaceMembers(workspaceId);
  const addMember = useAddChannelMember(channelId);
  const [addingUserId, setAddingUserId] = useState<string | null>(null);

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

  const getRoleBadge = (role?: string) => {
    if (!role || role === 'poster') return null;
    return (
      <span
        className={cn(
          'px-1.5 py-0.5 text-xs rounded',
          role === 'admin'
            ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
            : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
        )}
      >
        {role}
      </span>
    );
  };

  const renderMemberList = (membersList: ChannelMember[]) => (
    <div className="max-h-64 overflow-y-auto space-y-1">
      {membersList.map((member) => (
        <div
          key={member.user_id}
          className="flex items-center gap-3 px-3 py-2 rounded hover:bg-gray-50 dark:hover:bg-gray-700/50"
        >
          <Avatar
            src={member.avatar_url}
            name={member.display_name}
            size="sm"
          />
          <span className="flex-1 text-gray-900 dark:text-white">
            {member.display_name}
          </span>
          {getRoleBadge(member.channel_role)}
        </div>
      ))}
      {membersList.length === 0 && (
        <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
          No members found
        </p>
      )}
    </div>
  );

  const renderAddMemberList = (membersList: WorkspaceMemberWithUser[]) => (
    <div className="max-h-64 overflow-y-auto space-y-1">
      {membersList.map((member) => {
        const displayName = member.display_name_override || member.display_name;
        const isAdding = addingUserId === member.user_id;

        return (
          <div
            key={member.user_id}
            className="flex items-center gap-3 px-3 py-2 rounded hover:bg-gray-50 dark:hover:bg-gray-700/50"
          >
            <Avatar
              src={member.avatar_url}
              name={displayName}
              size="sm"
            />
            <span className="flex-1 text-gray-900 dark:text-white">
              {displayName}
            </span>
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
        <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
          All workspace members are already in this channel
        </p>
      )}
    </div>
  );

  const isLoading = membersLoading || (canAddMembers && workspaceMembersLoading);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Channel Members" size="md">
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Spinner />
        </div>
      ) : canAddMembers ? (
        <Tabs>
          <TabList>
            <Tab id="members">Members ({members.length})</Tab>
            <Tab id="add">Add Members</Tab>
          </TabList>
          <TabPanel id="members" className="pt-4">
            {renderMemberList(members)}
          </TabPanel>
          <TabPanel id="add" className="pt-4">
            {renderAddMemberList(nonMembers)}
          </TabPanel>
        </Tabs>
      ) : (
        renderMemberList(members)
      )}
    </Modal>
  );
}
