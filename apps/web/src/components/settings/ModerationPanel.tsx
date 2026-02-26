import { useState } from 'react';
import { ShieldExclamationIcon } from '@heroicons/react/24/outline';
import { Avatar, Button, Modal, Spinner, Tabs, TabList, Tab, TabPanel, toast } from '../ui';
import {
  useBans,
  useBanUser,
  useUnbanUser,
  useModerationLog,
} from '../../hooks/useModeration';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';

interface ModerationPanelProps {
  workspaceId: string;
}

export function ModerationPanel({ workspaceId }: ModerationPanelProps) {
  const [subTab, setSubTab] = useState<'bans' | 'log'>('bans');

  return (
    <div className="space-y-4">
      <Tabs selectedKey={subTab} onSelectionChange={(key) => setSubTab(key as 'bans' | 'log')}>
        <TabList>
          <Tab id="bans">Banned Users</Tab>
          <Tab id="log">Audit Log</Tab>
        </TabList>

        <TabPanel id="bans" className="pt-4">
          <BansList workspaceId={workspaceId} />
        </TabPanel>

        <TabPanel id="log" className="pt-4">
          <AuditLog workspaceId={workspaceId} />
        </TabPanel>
      </Tabs>
    </div>
  );
}

function BansList({ workspaceId }: { workspaceId: string }) {
  const { data, isLoading } = useBans(workspaceId);
  const unbanUser = useUnbanUser(workspaceId);
  const [showBanModal, setShowBanModal] = useState(false);
  const [unbanningUserId, setUnbanningUserId] = useState<string | null>(null);

  const handleUnban = async (userId: string) => {
    setUnbanningUserId(userId);
    try {
      await unbanUser.mutateAsync(userId);
      toast('User unbanned', 'success');
    } catch {
      toast('Failed to unban user', 'error');
    } finally {
      setUnbanningUserId(null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner size="md" />
      </div>
    );
  }

  const bans = data?.bans ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          {bans.length === 0 ? 'No banned users.' : `${bans.length} banned user${bans.length !== 1 ? 's' : ''}`}
        </p>
        <Button size="sm" onPress={() => setShowBanModal(true)}>
          Ban User
        </Button>
      </div>

      {bans.length > 0 && (
        <div className="space-y-3">
          {bans.map((ban) => (
            <div
              key={ban.id}
              className="flex items-center justify-between rounded-lg bg-gray-50 p-4 dark:bg-gray-800"
            >
              <div className="flex items-center gap-3">
                <Avatar
                  name={ban.user_display_name || 'User'}
                  id={ban.user_id}
                  size="md"
                />
                <div>
                  <p className="font-medium text-gray-900 dark:text-white">
                    {ban.user_display_name || 'Unknown User'}
                  </p>
                  {ban.reason && (
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      Reason: {ban.reason}
                    </p>
                  )}
                  <p className="text-xs text-gray-400 dark:text-gray-500">
                    Banned {new Date(ban.created_at).toLocaleDateString()}
                    {ban.expires_at && ` · Expires ${new Date(ban.expires_at).toLocaleDateString()}`}
                  </p>
                </div>
              </div>
              <Button
                variant="secondary"
                size="sm"
                onPress={() => handleUnban(ban.user_id)}
                isLoading={unbanningUserId === ban.user_id}
              >
                Unban
              </Button>
            </div>
          ))}
        </div>
      )}

      <BanUserModal
        isOpen={showBanModal}
        onClose={() => setShowBanModal(false)}
        workspaceId={workspaceId}
      />
    </div>
  );
}

function BanUserModal({
  isOpen,
  onClose,
  workspaceId,
}: {
  isOpen: boolean;
  onClose: () => void;
  workspaceId: string;
}) {
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const banUser = useBanUser(workspaceId);
  const [selectedUserId, setSelectedUserId] = useState('');
  const [reason, setReason] = useState('');
  const [duration, setDuration] = useState('');
  const [hideMessages, setHideMessages] = useState(false);

  const resetForm = () => {
    setSelectedUserId('');
    setReason('');
    setDuration('');
    setHideMessages(false);
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const handleBan = async () => {
    if (!selectedUserId) {
      toast('Select a user to ban', 'error');
      return;
    }

    try {
      await banUser.mutateAsync({
        user_id: selectedUserId,
        reason: reason || undefined,
        duration_hours: duration ? parseInt(duration) : undefined,
        hide_messages: hideMessages,
      });
      toast('User banned', 'success');
      handleClose();
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to ban user', 'error');
    }
  };

  const members = membersData?.members?.filter(
    (m) => m.role !== 'owner',
  ) ?? [];

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Ban User" size="sm">
      <div className="space-y-4">
        <div>
          <label htmlFor="ban-user-select" className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            User
          </label>
          <select
            id="ban-user-select"
            value={selectedUserId}
            onChange={(e) => setSelectedUserId(e.target.value)}
            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
          >
            <option value="">Select a user...</option>
            {members.map((m) => (
              <option key={m.user_id} value={m.user_id}>
                {m.display_name} ({m.role})
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="ban-reason" className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            Reason (optional)
          </label>
          <input
            id="ban-reason"
            type="text"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Reason for ban..."
            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
          />
        </div>

        <div>
          <label htmlFor="ban-duration" className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            Duration
          </label>
          <select
            id="ban-duration"
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
          >
            <option value="">Permanent</option>
            <option value="1">1 hour</option>
            <option value="24">24 hours</option>
            <option value="168">7 days</option>
            <option value="720">30 days</option>
          </select>
        </div>

        <label htmlFor="ban-hide-messages" className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            id="ban-hide-messages"
            type="checkbox"
            checked={hideMessages}
            onChange={(e) => setHideMessages(e.target.checked)}
            className="rounded border-gray-300 dark:border-gray-600"
          />
          Hide user's messages from other members
        </label>

        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" onPress={handleClose}>
            Cancel
          </Button>
          <Button
            variant="danger"
            onPress={handleBan}
            isLoading={banUser.isPending}
            isDisabled={!selectedUserId}
          >
            Ban User
          </Button>
        </div>
      </div>
    </Modal>
  );
}

function AuditLog({ workspaceId }: { workspaceId: string }) {
  const { data, isLoading, hasNextPage, fetchNextPage, isFetchingNextPage } =
    useModerationLog(workspaceId);

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner size="md" />
      </div>
    );
  }

  const entries = data?.pages.flatMap((page) => page.entries) ?? [];

  if (entries.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        No moderation actions recorded yet.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {entries.map((entry) => (
        <div
          key={entry.id}
          className="rounded-lg bg-gray-50 px-4 py-3 dark:bg-gray-800"
        >
          <div className="flex items-center gap-2">
            <ShieldExclamationIcon className="h-4 w-4 flex-shrink-0 text-gray-400" />
            <span className="text-sm text-gray-900 dark:text-white">
              <span className="font-medium">{entry.actor_display_name || 'System'}</span>
              {' '}
              {formatAction(entry.action)}
              {entry.target_display_name && (
                <>
                  {' '}
                  <span className="font-medium">{entry.target_display_name}</span>
                </>
              )}
            </span>
          </div>
          {entry.metadata && typeof entry.metadata === 'object' && Object.keys(entry.metadata).length > 0 && (
            <div className="mt-1 ml-6 text-xs text-gray-500 dark:text-gray-400">
              {Object.entries(entry.metadata).map(([key, value]) => (
                <span key={key} className="mr-3">
                  {formatMetadataKey(key)}: {String(value)}
                </span>
              ))}
            </div>
          )}
          <p className="mt-1 ml-6 text-xs text-gray-400 dark:text-gray-500">
            {new Date(entry.created_at).toLocaleString()}
          </p>
        </div>
      ))}

      {hasNextPage && (
        <div className="flex justify-center pt-2">
          <Button
            variant="secondary"
            size="sm"
            onPress={() => fetchNextPage()}
            isLoading={isFetchingNextPage}
          >
            Load More
          </Button>
        </div>
      )}
    </div>
  );
}

function formatMetadataKey(key: string): string {
  return key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatAction(action: string): string {
  switch (action) {
    case 'user.banned':
      return 'banned a user';
    case 'user.unbanned':
      return 'unbanned a user';
    case 'member.removed':
      return 'removed a member';
    case 'member.role_changed':
      return 'changed a member role';
    case 'message.deleted':
      return 'deleted a message';
    case 'channel.archived':
      return 'archived a channel';
    default:
      return action;
  }
}
