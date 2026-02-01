import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { useWorkspace, useWorkspaceMembers, useUpdateMemberRole, useRemoveMember } from '../hooks/useWorkspaces';
import { useAuth } from '../hooks';
import { Avatar, Button, Spinner, toast } from '../components/ui';
import { cn } from '../lib/utils';
import type { WorkspaceRole } from '@feather/api-client';

export function WorkspaceSettingsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { user } = useAuth();
  const { data: workspace, isLoading: workspaceLoading } = useWorkspace(workspaceId);
  const { data: membersData, isLoading: membersLoading } = useWorkspaceMembers(workspaceId);
  const updateRole = useUpdateMemberRole(workspaceId!);
  const removeMember = useRemoveMember(workspaceId!);

  const [activeTab, setActiveTab] = useState<'general' | 'members'>('general');

  const members = membersData?.members ?? [];
  const isLoading = workspaceLoading || membersLoading;

  const handleRoleChange = async (userId: string, role: WorkspaceRole) => {
    try {
      await updateRole.mutateAsync({ userId, role });
      toast('Role updated', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to update role', 'error');
    }
  };

  const handleRemoveMember = async (userId: string) => {
    if (!confirm('Are you sure you want to remove this member?')) return;
    try {
      await removeMember.mutateAsync(userId);
      toast('Member removed', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to remove member', 'error');
    }
  };

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto bg-white dark:bg-gray-900">
      <div className="max-w-4xl mx-auto p-8">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Workspace Settings
          </h1>
          <Link
            to={`/workspaces/${workspaceId}`}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <XMarkIcon className="w-6 h-6" />
          </Link>
        </div>

        {/* Tabs */}
        <div className="border-b border-gray-200 dark:border-gray-700 mb-6">
          <nav className="flex gap-4">
            <button
              onClick={() => setActiveTab('general')}
              className={cn(
                'py-2 px-1 border-b-2 font-medium text-sm',
                activeTab === 'general'
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
              )}
            >
              General
            </button>
            <button
              onClick={() => setActiveTab('members')}
              className={cn(
                'py-2 px-1 border-b-2 font-medium text-sm',
                activeTab === 'members'
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
              )}
            >
              Members ({members.length})
            </button>
          </nav>
        </div>

        {activeTab === 'general' && (
          <div className="space-y-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Workspace Name
              </label>
              <p className="text-gray-900 dark:text-white">{workspace?.workspace.name}</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Workspace URL
              </label>
              <p className="text-gray-900 dark:text-white">{workspace?.workspace.slug}</p>
            </div>
          </div>
        )}

        {activeTab === 'members' && (
          <div className="space-y-4">
            {members.map((member) => (
              <div
                key={member.user_id}
                className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg"
              >
                <div className="flex items-center gap-3">
                  <Avatar
                    src={member.avatar_url}
                    name={member.display_name || 'User'}
                    size="md"
                  />
                  <div>
                    <p className="font-medium text-gray-900 dark:text-white">
                      {member.display_name}
                      {member.user_id === user?.id && (
                        <span className="ml-2 text-xs text-gray-500">(you)</span>
                      )}
                    </p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {member.email}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <select
                    value={member.role}
                    onChange={(e) => handleRoleChange(member.user_id, e.target.value as WorkspaceRole)}
                    disabled={member.user_id === user?.id}
                    className="text-sm bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded px-2 py-1 text-gray-900 dark:text-white disabled:opacity-50"
                  >
                    <option value="owner">Owner</option>
                    <option value="admin">Admin</option>
                    <option value="member">Member</option>
                  </select>

                  {member.user_id !== user?.id && (
                    <Button
                      variant="secondary"
                      size="sm"
                      onPress={() => handleRemoveMember(member.user_id)}
                    >
                      Remove
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
