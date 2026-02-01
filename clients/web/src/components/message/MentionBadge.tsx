import { useState } from 'react';
import { Button as AriaButton, Popover, DialogTrigger } from 'react-aria-components';
import { tv } from 'tailwind-variants';
import { Avatar } from '../ui';
import { useProfilePanel } from '../../hooks/usePanel';
import type { WorkspaceMemberWithUser } from '@feather/api-client';

const mentionBadge = tv({
  base: [
    'inline rounded px-0.5 -mx-0.5 cursor-pointer',
    'text-blue-600 dark:text-blue-400',
    'bg-blue-50 dark:bg-blue-900/30',
    'hover:bg-blue-100 dark:hover:bg-blue-900/50',
    'transition-colors',
  ],
});

const popoverStyles = tv({
  slots: {
    popover: [
      'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700',
      'rounded-lg shadow-lg p-3 min-w-[200px]',
      'entering:animate-in entering:fade-in entering:zoom-in-95 entering:duration-150',
      'exiting:animate-out exiting:fade-out exiting:zoom-out-95 exiting:duration-100',
    ],
    header: 'flex items-center gap-3',
    info: 'flex flex-col',
    name: 'font-medium text-gray-900 dark:text-white',
    email: 'text-sm text-gray-500 dark:text-gray-400',
    viewProfile: [
      'mt-3 w-full text-center text-sm py-1.5 rounded',
      'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600',
      'text-gray-700 dark:text-gray-300',
      'transition-colors cursor-pointer',
    ],
  },
});

interface UserMentionBadgeProps {
  userId: string;
  member?: WorkspaceMemberWithUser;
}

export function UserMentionBadge({ userId, member }: UserMentionBadgeProps) {
  const [isOpen, setIsOpen] = useState(false);
  const { openProfile } = useProfilePanel();
  const styles = popoverStyles();

  const displayName = member?.display_name || 'Unknown User';

  const handleViewProfile = () => {
    setIsOpen(false);
    openProfile(userId);
  };

  return (
    <DialogTrigger isOpen={isOpen} onOpenChange={setIsOpen}>
      <AriaButton className={mentionBadge()}>
        @{displayName}
      </AriaButton>
      <Popover placement="top" className={styles.popover()}>
        <div className={styles.header()}>
          <Avatar
            src={member?.avatar_url}
            name={displayName}
            size="lg"
          />
          <div className={styles.info()}>
            <span className={styles.name()}>{displayName}</span>
            {member?.email && (
              <span className={styles.email()}>{member.email}</span>
            )}
          </div>
        </div>
        <button
          type="button"
          onClick={handleViewProfile}
          className={styles.viewProfile()}
        >
          View profile
        </button>
      </Popover>
    </DialogTrigger>
  );
}

interface SpecialMentionBadgeProps {
  type: string; // 'here' | 'channel' | 'everyone'
}

export function SpecialMentionBadge({ type }: SpecialMentionBadgeProps) {
  const descriptions: Record<string, string> = {
    here: 'Notifies everyone who is currently online',
    channel: 'Notifies everyone in this channel',
    everyone: 'Notifies everyone in this workspace',
  };

  return (
    <DialogTrigger>
      <AriaButton className={mentionBadge()}>
        @{type}
      </AriaButton>
      <Popover placement="top" className={popoverStyles().popover()}>
        <div className="text-sm">
          <div className="font-medium text-gray-900 dark:text-white mb-1">
            @{type}
          </div>
          <div className="text-gray-500 dark:text-gray-400">
            {descriptions[type] || 'Special mention'}
          </div>
        </div>
      </Popover>
    </DialogTrigger>
  );
}
