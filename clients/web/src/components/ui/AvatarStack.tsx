import { Avatar } from './Avatar';
import { cn } from '../../lib/utils';

interface AvatarStackUser {
  user_id: string;
  display_name?: string;
  avatar_url?: string | null;
}

interface AvatarStackProps {
  users: AvatarStackUser[];
  max?: number;
  size?: 'xs' | 'sm';
  showCount?: boolean;
  className?: string;
}

export function AvatarStack({
  users,
  max = 3,
  size = 'xs',
  showCount = true,
  className,
}: AvatarStackProps) {
  const visibleUsers = users.slice(0, max);
  const remainingCount = users.length - max;

  return (
    <div className={cn('flex -space-x-1.5', className)}>
      {visibleUsers.map((user, index) => (
        <Avatar
          key={user.user_id}
          src={user.avatar_url}
          name={user.display_name || 'User'}
          size={size}
          className="ring-2 ring-white dark:ring-gray-900"
          style={{ zIndex: visibleUsers.length - index }}
        />
      ))}
      {showCount && remainingCount > 0 && (
        <div
          className={cn(
            'flex items-center justify-center rounded-full bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300 font-medium ring-2 ring-white dark:ring-gray-900',
            size === 'xs' ? 'w-5 h-5 text-[9px]' : 'w-6 h-6 text-[10px]'
          )}
        >
          +{remainingCount}
        </div>
      )}
    </div>
  );
}
