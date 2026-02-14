import { useState } from 'react';
import { LockClosedIcon, HashtagIcon } from '@heroicons/react/24/outline';
import { Modal, Button, toast } from '../ui';
import { useConvertGroupDMToChannel } from '../../hooks/useChannels';

interface ConvertToChannelModalProps {
  isOpen: boolean;
  onClose: () => void;
  workspaceId: string;
  channelId: string;
}

export function ConvertToChannelModal({
  isOpen,
  onClose,
  workspaceId,
  channelId,
}: ConvertToChannelModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [type, setType] = useState<'public' | 'private'>('private');

  const convertGroupDM = useConvertGroupDMToChannel(workspaceId, channelId);

  const handleConvert = async () => {
    const trimmedName = name.trim();
    if (!trimmedName || !/^[a-z0-9]+(-[a-z0-9]+)*$/.test(trimmedName)) return;
    try {
      await convertGroupDM.mutateAsync({
        name: trimmedName,
        description: description.trim() || undefined,
        type,
      });
      toast('Conversation converted to channel', 'success');
      onClose();
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to convert', 'error');
    }
  };

  const handleClose = () => {
    setName('');
    setDescription('');
    setType('private');
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Convert to channel">
      <p className="mb-4 text-sm text-gray-600 dark:text-gray-300">
        Convert this group conversation into a channel. All messages will be preserved and all
        current members will remain.
      </p>
      <div className="mb-3">
        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Channel name
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value.toLowerCase().replace(/\s+/g, '-'))}
          placeholder="e.g. project-x"
          className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-transparent focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
        />
      </div>
      <div className="mb-3">
        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Visibility
        </label>
        <div className="flex gap-3">
          <label className="flex cursor-pointer items-center gap-2 rounded-md border border-gray-300 px-3 py-2 text-sm has-[:checked]:border-primary-500 has-[:checked]:ring-1 has-[:checked]:ring-primary-500 dark:border-gray-600 dark:has-[:checked]:border-primary-500">
            <input
              type="radio"
              name="convertType"
              value="private"
              checked={type === 'private'}
              onChange={() => setType('private')}
              className="accent-primary-500"
            />
            <LockClosedIcon className="h-4 w-4 text-gray-500" />
            <span className="text-gray-900 dark:text-white">Private</span>
          </label>
          <label className="flex cursor-pointer items-center gap-2 rounded-md border border-gray-300 px-3 py-2 text-sm has-[:checked]:border-primary-500 has-[:checked]:ring-1 has-[:checked]:ring-primary-500 dark:border-gray-600 dark:has-[:checked]:border-primary-500">
            <input
              type="radio"
              name="convertType"
              value="public"
              checked={type === 'public'}
              onChange={() => setType('public')}
              className="accent-primary-500"
            />
            <HashtagIcon className="h-4 w-4 text-gray-500" />
            <span className="text-gray-900 dark:text-white">Public</span>
          </label>
        </div>
      </div>
      <div className="mb-4">
        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Description (optional)
        </label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="What is this channel about?"
          className="w-full resize-none rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-transparent focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
          rows={2}
        />
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={handleClose}>
          Cancel
        </Button>
        <Button
          variant="primary"
          onClick={handleConvert}
          isLoading={convertGroupDM.isPending}
          isDisabled={!name.trim() || !/^[a-z0-9]+(-[a-z0-9]+)*$/.test(name.trim())}
        >
          Convert
        </Button>
      </div>
    </Modal>
  );
}
