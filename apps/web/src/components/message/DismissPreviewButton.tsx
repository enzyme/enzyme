import { XMarkIcon } from '@heroicons/react/24/outline';

interface DismissPreviewButtonProps {
  onDismiss: () => void;
  label?: string;
}

export function DismissPreviewButton({
  onDismiss,
  label = 'Remove link preview',
}: DismissPreviewButtonProps) {
  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onDismiss();
      }}
      className="absolute -top-2 -right-2 hidden cursor-pointer rounded-full border border-gray-200 bg-white p-0.5 text-gray-400 shadow-sm group-hover/preview:block hover:bg-gray-100 hover:text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-500 dark:hover:bg-gray-800 dark:hover:text-gray-300"
      aria-label={label}
    >
      <XMarkIcon className="h-3.5 w-3.5" />
    </button>
  );
}
