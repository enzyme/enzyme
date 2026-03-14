import { ChevronDownIcon } from '@heroicons/react/20/solid';
import { UnstyledButton } from '../ui';

interface JumpToLatestButtonProps {
  onClick: () => void;
  newMessageCount?: number;
}

export function JumpToLatestButton({ onClick, newMessageCount }: JumpToLatestButtonProps) {
  return (
    <UnstyledButton
      onPress={onClick}
      className="absolute bottom-4 left-1/2 z-10 flex -translate-x-1/2 items-center gap-1.5 rounded-full bg-white px-4 py-1.5 text-sm font-medium text-gray-700 shadow-lg ring-1 ring-gray-200 transition-colors hover:bg-gray-100 dark:bg-gray-800 dark:text-gray-200 dark:ring-gray-600 dark:hover:bg-gray-700"
    >
      <ChevronDownIcon className="h-4 w-4" />
      <span>Jump to latest</span>
      {newMessageCount != null && newMessageCount > 0 && (
        <span className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-blue-500 px-1.5 text-xs font-semibold text-white">
          {newMessageCount > 99 ? '99+' : newMessageCount}
        </span>
      )}
    </UnstyledButton>
  );
}
