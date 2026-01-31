import { cn, getInitials } from '../../lib/utils';
import type { PresenceStatus } from '@feather/api-client';

interface AvatarProps {
  src?: string | null;
  name: string;
  size?: 'sm' | 'md' | 'lg';
  status?: PresenceStatus;
  className?: string;
}

export function Avatar({ src, name, size = 'md', status, className }: AvatarProps) {
  const sizes = {
    sm: 'w-6 h-6 text-xs',
    md: 'w-8 h-8 text-sm',
    lg: 'w-10 h-10 text-base',
  };

  const statusColors = {
    online: 'bg-green-500',
    away: 'bg-yellow-500',
    offline: 'bg-gray-400',
  };

  const statusSizes = {
    sm: 'w-2 h-2 right-0 bottom-0',
    md: 'w-2.5 h-2.5 right-0 bottom-0',
    lg: 'w-3 h-3 right-0 bottom-0',
  };

  return (
    <div className={cn('relative inline-block', className)}>
      {src ? (
        <img
          src={src}
          alt={name}
          className={cn(
            'rounded-full object-cover',
            sizes[size]
          )}
        />
      ) : (
        <div
          className={cn(
            'rounded-full flex items-center justify-center font-medium',
            'bg-primary-500 text-white',
            sizes[size]
          )}
        >
          {getInitials(name)}
        </div>
      )}
      {status && (
        <span
          className={cn(
            'absolute rounded-full border-2 border-white dark:border-gray-900',
            statusColors[status],
            statusSizes[size]
          )}
        />
      )}
    </div>
  );
}
