import { useState } from 'react';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { useUserProfile, useUpdateProfile, useAuth } from '../../hooks';
import { useUIStore } from '../../stores/uiStore';
import { Avatar, Button, Input, Spinner, toast } from '../ui';
import { cn } from '../../lib/utils';

interface ProfilePaneProps {
  userId: string;
}

export function ProfilePane({ userId }: ProfilePaneProps) {
  const { closeProfile } = useUIStore();
  const { user: currentUser } = useAuth();
  const { data, isLoading } = useUserProfile(userId);
  const [isEditing, setIsEditing] = useState(false);

  const isOwnProfile = currentUser?.id === userId;
  const profile = data?.user;

  return (
    <div className="w-80 border-l border-gray-200 dark:border-gray-700 flex flex-col bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
        <h3 className="font-semibold text-gray-900 dark:text-white">Profile</h3>
        <button
          onClick={closeProfile}
          className="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded"
        >
          <XMarkIcon className="w-5 h-5" />
        </button>
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
              profile={profile}
              onCancel={() => setIsEditing(false)}
              onSuccess={() => setIsEditing(false)}
            />
          ) : (
            <ViewProfile
              profile={profile}
              isOwnProfile={isOwnProfile}
              onEdit={() => setIsEditing(true)}
            />
          )
        ) : (
          <div className="text-center text-gray-500 dark:text-gray-400 py-8">
            User not found
          </div>
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
    status: string;
    created_at: string;
  };
  isOwnProfile: boolean;
  onEdit: () => void;
}

function ViewProfile({ profile, isOwnProfile, onEdit }: ViewProfileProps) {
  const memberSince = new Date(profile.created_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
  });

  return (
    <div className="space-y-6">
      {/* Avatar and name */}
      <div className="flex flex-col items-center text-center">
        <Avatar
          src={profile.avatar_url}
          name={profile.display_name}
          size="lg"
          className="w-24 h-24 text-3xl"
        />
        <h4 className="mt-4 text-xl font-semibold text-gray-900 dark:text-white">
          {profile.display_name}
        </h4>
        <span className={cn(
          'mt-1 inline-flex items-center gap-1.5 text-sm',
          profile.status === 'active' ? 'text-green-600 dark:text-green-400' : 'text-gray-500'
        )}>
          <span className={cn(
            'w-2 h-2 rounded-full',
            profile.status === 'active' ? 'bg-green-500' : 'bg-gray-400'
          )} />
          {profile.status === 'active' ? 'Active' : profile.status}
        </span>
      </div>

      {/* Details */}
      <div className="space-y-3">
        <div>
          <dt className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
            Member since
          </dt>
          <dd className="mt-1 text-sm text-gray-900 dark:text-white">
            {memberSince}
          </dd>
        </div>
      </div>

      {/* Edit button (only for own profile) */}
      {isOwnProfile && (
        <Button
          variant="secondary"
          className="w-full"
          onPress={onEdit}
        >
          Edit Profile
        </Button>
      )}
    </div>
  );
}

interface EditProfileFormProps {
  profile: {
    display_name: string;
    avatar_url?: string;
  };
  onCancel: () => void;
  onSuccess: () => void;
}

function EditProfileForm({ profile, onCancel, onSuccess }: EditProfileFormProps) {
  const [displayName, setDisplayName] = useState(profile.display_name);
  const [avatarUrl, setAvatarUrl] = useState(profile.avatar_url || '');
  const updateProfile = useUpdateProfile();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const trimmedName = displayName.trim();
    if (!trimmedName) {
      toast('Display name is required', 'error');
      return;
    }

    try {
      await updateProfile.mutateAsync({
        display_name: trimmedName,
        avatar_url: avatarUrl.trim() || undefined,
      });
      toast('Profile updated', 'success');
      onSuccess();
    } catch {
      toast('Failed to update profile', 'error');
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Avatar preview */}
      <div className="flex flex-col items-center">
        <Avatar
          src={avatarUrl || undefined}
          name={displayName || 'User'}
          size="lg"
          className="w-24 h-24 text-3xl"
        />
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

        <Input
          label="Avatar URL"
          value={avatarUrl}
          onChange={(e) => setAvatarUrl(e.target.value)}
          placeholder="https://example.com/avatar.jpg"
          type="url"
        />
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <Button
          type="button"
          variant="secondary"
          className="flex-1"
          onPress={onCancel}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          className="flex-1"
          isLoading={updateProfile.isPending}
        >
          Save
        </Button>
      </div>
    </form>
  );
}
