import { useState } from 'react';
import { cn } from '../../lib/utils';
import { getInitials, getAvatarColor } from '@enzyme/shared';
import type { PresenceStatus } from '@enzyme/api-client';

// Module-level cache for gravatar URLs that returned 404
const failedGravatarUrls = new Set<string>();

interface AvatarProps {
  src?: string | null;
  gravatarSrc?: string | null;
  name: string;
  id?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg';
  status?: PresenceStatus;
  className?: string;
  style?: React.CSSProperties;
  onClick?: () => void;
  onContextMenu?: (e: React.MouseEvent) => void;
}

export function Avatar({
  src,
  gravatarSrc,
  name,
  id,
  size = 'md',
  status,
  className,
  style,
  onClick,
  onContextMenu,
}: AvatarProps) {
  const gravatarFailed = !!gravatarSrc && failedGravatarUrls.has(gravatarSrc);
  const [, forceUpdate] = useState(0);

  const sizes = {
    xs: 'w-5 h-5 text-[10px]',
    sm: 'w-6 h-6 text-xs',
    md: 'w-8 h-8 text-sm',
    lg: 'w-10 h-10 text-base',
  };

  const statusColors = {
    online: 'bg-green-500',
    offline: 'bg-gray-400',
  };

  const statusSizes = {
    xs: 'w-2 h-2 right-[-1px] bottom-[-1px] border-2',
    sm: 'w-2 h-2 right-0 bottom-0 border-2',
    md: 'w-2.5 h-2.5 right-0 bottom-0 border-2',
    lg: 'w-3 h-3 right-0 bottom-0 border-2',
  };

  const showGravatar = !src && gravatarSrc && !gravatarFailed;

  const content = (
    <>
      {src ? (
        <img src={src} alt={name} className={cn('rounded-full object-cover', sizes[size])} />
      ) : showGravatar ? (
        <img
          src={gravatarSrc}
          alt={name}
          className={cn('rounded-full object-cover', sizes[size])}
          onError={() => {
            failedGravatarUrls.add(gravatarSrc);
            forceUpdate((n) => n + 1);
          }}
        />
      ) : (
        <div
          className={cn(
            'flex items-center justify-center rounded-full font-medium text-white',
            id ? getAvatarColor(id) : 'bg-blue-500',
            sizes[size],
          )}
        >
          {getInitials(name)}
        </div>
      )}
      {status && (
        <span
          className={cn(
            'absolute rounded-full border-[var(--avatar-ring)]',
            statusColors[status],
            statusSizes[size],
          )}
        />
      )}
    </>
  );

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        onContextMenu={onContextMenu}
        className={cn('relative inline-block cursor-pointer rounded-full', className)}
        style={style}
      >
        {content}
      </button>
    );
  }

  return (
    <div
      className={cn('relative inline-block rounded-full', className)}
      style={style}
      onContextMenu={onContextMenu}
    >
      {content}
    </div>
  );
}
