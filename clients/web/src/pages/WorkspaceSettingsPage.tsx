import { useState, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import { XMarkIcon, PhotoIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useWorkspace, useWorkspaceMembers, useUpdateMemberRole, useRemoveMember, useUploadWorkspaceIcon, useDeleteWorkspaceIcon } from '../hooks/useWorkspaces';
import { useAuth } from '../hooks';
import { Avatar, Button, Spinner, toast } from '../components/ui';
import { cn, getAvatarColor } from '../lib/utils';
import type { WorkspaceRole } from '@feather/api-client';

export function WorkspaceSettingsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { user } = useAuth();
  const { data: workspace, isLoading: workspaceLoading } = useWorkspace(workspaceId);
  const { data: membersData, isLoading: membersLoading } = useWorkspaceMembers(workspaceId);
  const updateRole = useUpdateMemberRole(workspaceId!);
  const removeMember = useRemoveMember(workspaceId!);
  const uploadIcon = useUploadWorkspaceIcon(workspaceId!);
  const deleteIcon = useDeleteWorkspaceIcon(workspaceId!);

  const [activeTab, setActiveTab] = useState<'general' | 'members'>('general');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

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

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      toast('Invalid file type. Please use JPEG, PNG, GIF, or WebP.', 'error');
      return;
    }

    if (file.size > 5 * 1024 * 1024) {
      toast('File too large. Maximum size is 5MB.', 'error');
      return;
    }

    setSelectedFile(file);
    setPreviewUrl(URL.createObjectURL(file));
  };

  const handleClearSelection = () => {
    setSelectedFile(null);
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setPreviewUrl(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleUploadIcon = async () => {
    if (!selectedFile) return;
    try {
      await uploadIcon.mutateAsync(selectedFile);
      toast('Workspace icon updated', 'success');
      handleClearSelection();
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to upload icon', 'error');
    }
  };

  const handleRemoveIcon = async () => {
    try {
      await deleteIcon.mutateAsync();
      toast('Workspace icon removed', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to remove icon', 'error');
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
            {/* Workspace Icon */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                Workspace Icon
              </label>
              <div className="flex items-start gap-4">
                {/* Icon preview */}
                <div
                  className={cn(
                    'w-20 h-20 rounded-xl flex items-center justify-center overflow-hidden',
                    !previewUrl && !workspace?.workspace.icon_url && workspaceId && getAvatarColor(workspaceId)
                  )}
                >
                  {previewUrl || workspace?.workspace.icon_url ? (
                    <img
                      src={previewUrl || workspace?.workspace.icon_url}
                      alt={workspace?.workspace.name}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <span className="text-white font-bold text-2xl">
                      {workspace?.workspace.name?.slice(0, 2).toUpperCase()}
                    </span>
                  )}
                </div>

                {/* Upload controls */}
                <div className="flex flex-col gap-2">
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept="image/jpeg,image/png,image/gif,image/webp"
                    onChange={handleFileSelect}
                    className="hidden"
                  />

                  {selectedFile ? (
                    <>
                      <p className="text-sm text-gray-500 dark:text-gray-400 mb-1">
                        Selected: {selectedFile.name}
                      </p>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          onPress={handleUploadIcon}
                          isLoading={uploadIcon.isPending}
                        >
                          <PhotoIcon className="w-4 h-4 mr-1" />
                          Save Icon
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          onPress={handleClearSelection}
                          isDisabled={uploadIcon.isPending}
                        >
                          <XMarkIcon className="w-4 h-4 mr-1" />
                          Cancel
                        </Button>
                      </div>
                    </>
                  ) : (
                    <div className="flex gap-2">
                      <Button
                        variant="secondary"
                        size="sm"
                        onPress={() => fileInputRef.current?.click()}
                      >
                        <PhotoIcon className="w-4 h-4 mr-1" />
                        Upload Icon
                      </Button>
                      {workspace?.workspace.icon_url && (
                        <Button
                          variant="secondary"
                          size="sm"
                          onPress={handleRemoveIcon}
                          isLoading={deleteIcon.isPending}
                        >
                          <TrashIcon className="w-4 h-4 mr-1" />
                          Remove
                        </Button>
                      )}
                    </div>
                  )}

                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    JPEG, PNG, GIF, or WebP. Max 5MB.
                  </p>
                </div>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Workspace Name
              </label>
              <p className="text-gray-900 dark:text-white">{workspace?.workspace.name}</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Workspace ID
              </label>
              <p className="text-gray-900 dark:text-white">{workspace?.workspace.id}</p>
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
                    id={member.user_id}
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
