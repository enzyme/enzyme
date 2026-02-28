import { useState, useRef } from 'react';
import { TrashIcon } from '@heroicons/react/24/outline';
import {
  useCustomEmojis,
  useUploadCustomEmoji,
  useDeleteCustomEmoji,
} from '../../hooks/useCustomEmojis';
import { useAuth } from '../../hooks';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { Button, Spinner, toast, CustomEmojiImg, ConfirmDialog } from '../ui';
import { resolveStandardShortcode } from '../../lib/emoji';

const NAME_REGEX = /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$/;

interface CustomEmojiManagerProps {
  workspaceId: string;
}

export function CustomEmojiManager({ workspaceId }: CustomEmojiManagerProps) {
  const { user } = useAuth();
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const { data: emojis, isLoading } = useCustomEmojis(workspaceId);
  const uploadEmoji = useUploadCustomEmoji(workspaceId);
  const deleteEmoji = useDeleteCustomEmoji(workspaceId);

  const [name, setName] = useState('');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [emojiToDelete, setEmojiToDelete] = useState<{ id: string; name: string } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const currentMember = membersData?.members.find((m) => m.user_id === user?.id);
  const isAdmin = currentMember?.role === 'owner' || currentMember?.role === 'admin';

  const nameError = (() => {
    if (!name) return null;
    const lower = name.toLowerCase();
    if (!NAME_REGEX.test(name)) {
      return 'Name must start with a letter or number and contain only letters, numbers, hyphens, and underscores (max 63 chars).';
    }
    if (resolveStandardShortcode(lower)) {
      return `":${lower}:" is already a standard emoji. Choose a different name.`;
    }
    if (emojis?.some((e) => e.name === lower)) {
      return `":${lower}:" already exists in this workspace.`;
    }
    return null;
  })();

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const allowedTypes = ['image/png', 'image/gif'];
    if (!allowedTypes.includes(file.type)) {
      toast('Only PNG and GIF files are allowed.', 'error');
      return;
    }

    if (file.size > 256 * 1024) {
      toast('File too large. Maximum size is 256KB.', 'error');
      return;
    }

    setSelectedFile(file);
    setPreviewUrl(URL.createObjectURL(file));
  };

  const handleClear = () => {
    setSelectedFile(null);
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    setPreviewUrl(null);
    setName('');
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const handleUpload = async () => {
    if (!selectedFile || !name || nameError) return;
    try {
      await uploadEmoji.mutateAsync({ file: selectedFile, name: name.toLowerCase() });
      toast(`Emoji :${name.toLowerCase()}: added`, 'success');
      handleClear();
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to upload emoji', 'error');
    }
  };

  const handleDelete = async () => {
    if (!emojiToDelete) return;
    try {
      await deleteEmoji.mutateAsync(emojiToDelete.id);
      toast(`Emoji :${emojiToDelete.name}: deleted`, 'success');
      setEmojiToDelete(null);
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to delete emoji', 'error');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Upload form */}
      <div className="space-y-4 rounded-lg bg-gray-50 p-4 dark:bg-gray-800">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Add Custom Emoji</h3>

        <div className="flex items-start gap-4">
          {/* Preview */}
          <div className="flex h-16 w-16 flex-shrink-0 items-center justify-center overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-600 dark:bg-gray-700">
            {previewUrl ? (
              <img
                src={previewUrl}
                alt="Preview"
                className="max-h-full max-w-full object-contain"
              />
            ) : (
              <span className="text-xs text-gray-400">Preview</span>
            )}
          </div>

          <div className="flex-1 space-y-3">
            {/* File input */}
            <div>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/png,image/gif"
                onChange={handleFileSelect}
                className="hidden"
              />
              <Button variant="secondary" size="sm" onPress={() => fileInputRef.current?.click()}>
                {selectedFile ? 'Change Image' : 'Choose Image'}
              </Button>
              {selectedFile && (
                <span className="ml-2 text-sm text-gray-500 dark:text-gray-400">
                  {selectedFile.name}
                </span>
              )}
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                PNG or GIF. Max 256KB.
              </p>
            </div>

            {/* Name input */}
            <div>
              <label className="mb-1 block text-sm text-gray-600 dark:text-gray-400">Name</label>
              <div className="flex items-center gap-2">
                <span className="text-gray-400">:</span>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value.replace(/\s/g, ''))}
                  placeholder="emoji_name"
                  className="flex-1 rounded border border-gray-300 bg-white px-2 py-1 text-sm text-gray-900 placeholder-gray-400 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                />
                <span className="text-gray-400">:</span>
              </div>
              {nameError && <p className="mt-1 text-xs text-red-500">{nameError}</p>}
            </div>

            {/* Upload button */}
            <Button
              size="sm"
              onPress={handleUpload}
              isDisabled={!selectedFile || !name || !!nameError}
              isLoading={uploadEmoji.isPending}
            >
              Upload Emoji
            </Button>
          </div>
        </div>
      </div>

      {/* Emoji list */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
          Custom Emojis ({emojis?.length || 0})
        </h3>

        {emojis && emojis.length > 0 ? (
          <div className="space-y-2">
            {emojis.map((emoji) => (
              <div
                key={emoji.id}
                className="flex items-center justify-between rounded-lg bg-gray-50 p-3 dark:bg-gray-800"
              >
                <div className="flex items-center gap-3">
                  <CustomEmojiImg name={emoji.name} url={emoji.url} size="lg" />
                  <div>
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      :{emoji.name}:
                    </span>
                  </div>
                </div>

                {(emoji.created_by === user?.id || isAdmin) && (
                  <Button
                    variant="secondary"
                    size="sm"
                    onPress={() => setEmojiToDelete({ id: emoji.id, name: emoji.name })}
                  >
                    <TrashIcon className="h-4 w-4" />
                  </Button>
                )}
              </div>
            ))}
          </div>
        ) : (
          <p className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
            No custom emojis yet. Upload one above to get started.
          </p>
        )}
      </div>

      {emojiToDelete && (
        <ConfirmDialog
          isOpen
          onClose={() => setEmojiToDelete(null)}
          onConfirm={handleDelete}
          title="Delete emoji"
          description={`Delete :${emojiToDelete.name}:? Messages using it will show the shortcode as plain text.`}
          confirmLabel="Delete"
          variant="destructive"
          isLoading={deleteEmoji.isPending}
        />
      )}
    </div>
  );
}
