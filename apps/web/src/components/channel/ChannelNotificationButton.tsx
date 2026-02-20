import { Button as AriaButton } from 'react-aria-components';
import { BellAlertIcon, BellIcon, BellSlashIcon } from '@heroicons/react/24/outline';
import { SelectMenu, SelectMenuItem, Spinner } from '../ui';
import { useChannelNotifications, useUpdateChannelNotifications } from '../../hooks';
import type { ChannelType, NotifyLevel } from '@enzyme/api-client';
import { cn } from '../../lib/utils';

interface ChannelNotificationButtonProps {
  channelId: string;
  channelType: ChannelType;
}

const NOTIFICATION_OPTIONS: { level: NotifyLevel; label: string; icon: typeof BellIcon }[] = [
  { level: 'all', label: 'All messages', icon: BellAlertIcon },
  { level: 'mentions', label: 'Mentions only', icon: BellIcon },
  { level: 'none', label: 'Muted', icon: BellSlashIcon },
];

export function ChannelNotificationButton({
  channelId,
  channelType,
}: ChannelNotificationButtonProps) {
  const { data, isLoading } = useChannelNotifications(channelId);
  const updateNotifications = useUpdateChannelNotifications(channelId);

  // Hide for DM channels
  if (channelType === 'dm' || channelType === 'group_dm') {
    return null;
  }

  const currentLevel = data?.preferences?.notify_level || 'all';
  const currentEmailEnabled = data?.preferences?.email_enabled ?? true;
  const isMuted = currentLevel === 'none';

  const handleSelect = (key: React.Key) => {
    updateNotifications.mutate({
      notify_level: key as NotifyLevel,
      email_enabled: currentEmailEnabled,
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center px-2">
        <Spinner size="sm" />
      </div>
    );
  }

  return (
    <SelectMenu
      selectedKey={currentLevel}
      onSelectionChange={handleSelect}
      trigger={
        <AriaButton
          className={cn(
            'cursor-pointer rounded p-1.5 outline-none hover:bg-gray-100 hover:text-gray-900 focus-visible:ring-2 focus-visible:ring-blue-500 dark:hover:bg-gray-800 dark:hover:text-gray-200',
            isMuted ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400',
          )}
          aria-label="Channel notification settings"
        >
          {currentLevel === 'none' ? (
            <BellSlashIcon className="h-4 w-4" />
          ) : currentLevel === 'mentions' ? (
            <BellIcon className="h-4 w-4" />
          ) : (
            <BellAlertIcon className="h-4 w-4" />
          )}
        </AriaButton>
      }
      align="end"
    >
      {NOTIFICATION_OPTIONS.map(({ level, label, icon: Icon }) => (
        <SelectMenuItem
          key={level}
          id={level}
          icon={<Icon className="h-4 w-4 text-gray-500 dark:text-gray-400" />}
        >
          {label}
        </SelectMenuItem>
      ))}
    </SelectMenu>
  );
}
