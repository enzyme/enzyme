import { useState, useRef } from 'react';
import { useParams } from 'react-router-dom';
import { XMarkIcon, PhotoIcon, TrashIcon, NoSymbolIcon } from '@heroicons/react/24/outline';
import {
  useUserProfile,
  useUpdateProfile,
  useUploadAvatar,
  useDeleteAvatar,
  useAuth,
} from '../../hooks';
import { useProfilePanel } from '../../hooks/usePanel';
import { useBlocks, useBlockUser, useUnblockUser } from '../../hooks/useModeration';
import { Button, IconButton, Input, Spinner, toast } from '../ui';
import { cn, getInitials, getAvatarColor } from '../../lib/utils';
import { useUserPresence } from '../../lib/presenceStore';

interface ProfilePaneProps {
  userId: string;
}

export function ProfilePane({ userId }: ProfilePaneProps) {
  const { closeProfile } = useProfilePanel();
  const { user: currentUser } = useAuth();
  const { data, isLoading } = useUserProfile(userId);
  const [isEditing, setIsEditing] = useState(false);

  const isOwnProfile = currentUser?.id === userId;
  const profile = data?.user;

  return (
    <div className="flex w-80 flex-col border-l border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-gray-200 p-3 dark:border-gray-700">
        <h3 className="font-semibold text-gray-900 dark:text-white">Profile</h3>
        <IconButton onPress={closeProfile} aria-label="Close profile">
          <XMarkIcon className="h-4 w-4" />
        </IconButton>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {isLoading ? (
          <div className="flex justify-center py-8">
            <Spinner size="lg" />
          </div>
        ) : profile ? (
          isEditing && isOwnProfile ? (
            <EditProfileForm
              userId={userId}
              profile={profile}
              onCancel={() => setIsEditing(false)}
              onSuccess={() => setIsEditing(false)}
            />
          ) : (
            <ViewProfile
              key={profile.id}
              profile={profile}
              isOwnProfile={isOwnProfile}
              onEdit={() => setIsEditing(true)}
            />
          )
        ) : (
          <div className="py-8 text-center text-gray-500 dark:text-gray-400">User not found</div>
        )}
      </div>
    </div>
  );
}

interface ViewProfileProps {
  profile: {
    id: string;
    display_name: string;
    avatar_url?: string;
    gravatar_url?: string;
    status: string;
    created_at: string;
  };
  isOwnProfile: boolean;
  onEdit: () => void;
}

function ViewProfile({ profile, isOwnProfile, onEdit }: ViewProfileProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const [gravatarFailed, setGravatarFailed] = useState(false);
  const presence = useUserPresence(profile.id);
  const { data: blocksData } = useBlocks(workspaceId);
  const blockUserMutation = useBlockUser(workspaceId);
  const unblockUserMutation = useUnblockUser(workspaceId);
  const isBlocked = blocksData?.blocks?.some((b) => b.blocked_id === profile.id) ?? false;

  const handleToggleBlock = async () => {
    try {
      if (isBlocked) {
        await unblockUserMutation.mutateAsync(profile.id);
        toast('User unblocked', 'success');
      } else {
        await blockUserMutation.mutateAsync(profile.id);
        toast('User blocked', 'success');
      }
    } catch {
      toast(isBlocked ? 'Failed to unblock user' : 'Failed to block user', 'error');
    }
  };
  const memberSince = new Date(profile.created_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
  });

  const statusConfig = {
    online: { color: 'text-green-600 dark:text-green-400', dot: 'bg-green-500', label: 'Online' },
    offline: { color: 'text-gray-500', dot: 'bg-gray-400', label: 'Offline' },
  };
  const status = statusConfig[presence ?? 'offline'];

  const bannerUrl = profile.avatar_url || (!gravatarFailed ? profile.gravatar_url : undefined);

  return (
    <div className="space-y-6">
      {/* Profile banner */}
      {bannerUrl ? (
        <div className="flex justify-center pt-2">
          <img
            src={bannerUrl}
            alt={profile.display_name}
            className="w-full rounded-lg"
            onError={!profile.avatar_url ? () => setGravatarFailed(true) : undefined}
          />
        </div>
      ) : (
        <div
          className={cn(
            'flex h-48 items-center justify-center rounded-lg text-5xl font-medium text-white',
            getAvatarColor(profile.id),
          )}
        >
          {getInitials(profile.display_name)}
        </div>
      )}

      {/* Name and status */}
      <div className="flex flex-col items-center text-center">
        <h4 className="text-xl font-semibold text-gray-900 dark:text-white">
          {profile.display_name}
        </h4>
        <span className={cn('mt-1 inline-flex items-center gap-1.5 text-sm', status.color)}>
          <span className={cn('h-2 w-2 rounded-full', status.dot)} />
          {status.label}
        </span>
      </div>

      {/* Details */}
      <div className="space-y-3">
        <div>
          <dt className="text-xs font-medium tracking-wider text-gray-500 uppercase dark:text-gray-400">
            Member since
          </dt>
          <dd className="mt-1 text-sm text-gray-900 dark:text-white">{memberSince}</dd>
        </div>
      </div>

      {/* Edit button (only for own profile) */}
      {isOwnProfile && (
        <Button variant="secondary" className="w-full" onPress={onEdit}>
          Edit Profile
        </Button>
      )}

      {/* Block button (only for other profiles) */}
      {!isOwnProfile && (
        <Button
          variant={isBlocked ? 'secondary' : 'danger'}
          className="w-full"
          onPress={handleToggleBlock}
          isLoading={blockUserMutation.isPending || unblockUserMutation.isPending}
        >
          <NoSymbolIcon className="mr-1 h-4 w-4" />
          {isBlocked ? 'Unblock User' : 'Block User'}
        </Button>
      )}
    </div>
  );
}

interface EditProfileFormProps {
  userId: string;
  profile: {
    display_name: string;
    avatar_url?: string;
    gravatar_url?: string;
  };
  onCancel: () => void;
  onSuccess: () => void;
}

function EditProfileForm({ userId, profile, onCancel, onSuccess }: EditProfileFormProps) {
  const [displayName, setDisplayName] = useState(profile.display_name);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const updateProfile = useUpdateProfile();
  const uploadAvatar = useUploadAvatar();
  const deleteAvatar = useDeleteAvatar();

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      toast('Invalid file type. Please use JPEG, PNG, GIF, or WebP.', 'error');
      return;
    }

    // Validate file size (5MB)
    if (file.size > 5 * 1024 * 1024) {
      toast('File too large. Maximum size is 5MB.', 'error');
      return;
    }

    setSelectedFile(file);
    setPreviewUrl(URL.createObjectURL(file));
  };

  const handleRemoveAvatar = async () => {
    try {
      await deleteAvatar.mutateAsync();
      setSelectedFile(null);
      setPreviewUrl(null);
      toast('Avatar removed', 'success');
    } catch {
      toast('Failed to remove avatar', 'error');
    }
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const trimmedName = displayName.trim();
    if (!trimmedName) {
      toast('Display name is required', 'error');
      return;
    }

    try {
      // Upload avatar file if selected
      if (selectedFile) {
        await uploadAvatar.mutateAsync(selectedFile);
      }

      // Update profile display name
      await updateProfile.mutateAsync({
        display_name: trimmedName,
      });

      toast('Profile updated', 'success');
      onSuccess();
    } catch {
      toast('Failed to update profile', 'error');
    }
  };

  const hasExistingAvatar = !!profile.avatar_url;
  const isPending = updateProfile.isPending || uploadAvatar.isPending || deleteAvatar.isPending;

  const displayAvatarUrl = previewUrl || profile.avatar_url || profile.gravatar_url;

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Profile banner */}
      {displayAvatarUrl ? (
        <div className="flex justify-center pt-2">
          <img src={displayAvatarUrl} alt={displayName || 'User'} className="w-full rounded-lg" />
        </div>
      ) : (
        <div
          className={cn(
            'flex h-48 items-center justify-center rounded-lg text-5xl font-medium text-white',
            getAvatarColor(userId),
          )}
        >
          {getInitials(displayName || 'User')}
        </div>
      )}

      {/* Avatar upload */}
      <div className="flex flex-col items-center gap-3">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          onChange={handleFileSelect}
          className="hidden"
        />

        <div className="flex flex-wrap justify-center gap-2">
          {selectedFile ? (
            <Button type="button" variant="secondary" size="sm" onPress={handleClearSelection}>
              <XMarkIcon className="mr-1 h-4 w-4" />
              Clear
            </Button>
          ) : (
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onPress={() => fileInputRef.current?.click()}
            >
              <PhotoIcon className="mr-1 h-4 w-4" />
              Upload Photo
            </Button>
          )}

          {hasExistingAvatar && !selectedFile && (
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onPress={handleRemoveAvatar}
              isLoading={deleteAvatar.isPending}
            >
              <TrashIcon className="mr-1 h-4 w-4" />
              Remove
            </Button>
          )}
        </div>

        {selectedFile && (
          <p className="text-xs text-gray-500 dark:text-gray-400">Selected: {selectedFile.name}</p>
        )}
      </div>

      {/* Form fields */}
      <div className="space-y-4">
        <Input
          label="Display Name"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          placeholder="Your name"
          isRequired
        />
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <Button
          type="button"
          variant="secondary"
          className="flex-1"
          onPress={onCancel}
          isDisabled={isPending}
        >
          Cancel
        </Button>
        <Button type="submit" className="flex-1" isLoading={isPending}>
          Save
        </Button>
      </div>
    </form>
  );
}
