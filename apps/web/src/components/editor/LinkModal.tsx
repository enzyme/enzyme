import { useState, useRef, useEffect } from 'react';
import { Modal, Input } from '../ui';
import { Button as AriaButton } from 'react-aria-components';

export interface LinkModalData {
  text: string;
  url: string;
  isEdit: boolean;
}

interface LinkModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (text: string, url: string) => void;
  initialData?: LinkModalData;
}

export function LinkModal({ isOpen, onClose, onSave, initialData }: LinkModalProps) {
  // Key resets the form whenever the modal opens with new data
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={initialData?.isEdit ? 'Edit Link' : 'Insert Link'}
      size="sm"
    >
      {isOpen && (
        <LinkForm
          key={initialData?.url ?? ''}
          initialText={initialData?.text ?? ''}
          initialUrl={initialData?.url ?? ''}
          onSave={onSave}
          onClose={onClose}
        />
      )}
    </Modal>
  );
}

function LinkForm({
  initialText,
  initialUrl,
  onSave,
  onClose,
}: {
  initialText: string;
  initialUrl: string;
  onSave: (text: string, url: string) => void;
  onClose: () => void;
}) {
  const [text, setText] = useState(initialText);
  const [url, setUrl] = useState(initialUrl);
  const urlInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const timer = setTimeout(() => {
      urlInputRef.current?.focus();
    }, 50);
    return () => clearTimeout(timer);
  }, []);

  const handleSave = () => {
    if (!url.trim()) return;
    onSave(text.trim(), url.trim());
    onClose();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSave();
    }
  };

  return (
    <div className="flex flex-col gap-4" onKeyDown={handleKeyDown}>
      <Input
        label="Text"
        placeholder="Display text"
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <div>
        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          URL
        </label>
        <input
          ref={urlInputRef}
          type="url"
          placeholder="https://example.com"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          className="w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm transition-colors placeholder:text-gray-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none dark:border-gray-600 dark:bg-gray-800 dark:text-white dark:placeholder:text-gray-500"
        />
      </div>
      <div className="flex justify-end gap-2">
        <AriaButton
          onPress={onClose}
          className="cursor-pointer rounded-md border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
        >
          Cancel
        </AriaButton>
        <AriaButton
          onPress={handleSave}
          isDisabled={!url.trim()}
          className="cursor-pointer rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Save
        </AriaButton>
      </div>
    </div>
  );
}
