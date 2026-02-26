import { XMarkIcon } from '@heroicons/react/24/outline';
import { IconButton } from '../ui';

interface DismissPreviewButtonProps {
  onDismiss: () => void;
  label?: string;
}

export function DismissPreviewButton({
  onDismiss,
  label = 'Remove link preview',
}: DismissPreviewButtonProps) {
  return (
    <IconButton
      onPress={onDismiss}
      aria-label={label}
      size="xs"
      className="absolute -top-2 -right-2 hidden rounded-full border border-gray-200 bg-white shadow-sm group-hover/preview:block hover:bg-gray-100 dark:border-gray-600 dark:bg-gray-900 dark:hover:bg-gray-800"
    >
      <XMarkIcon className="h-3.5 w-3.5" />
    </IconButton>
  );
}
