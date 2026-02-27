import { DialogTrigger, Popover } from 'react-aria-components';
import {
  FaceSmileIcon,
  ChatBubbleBottomCenterTextIcon,
  EllipsisVerticalIcon,
  LinkIcon,
  EyeSlashIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { IconButton, Tooltip, Menu, MenuItem, EmojiGrid, PinOutlineIcon } from '../ui';
import { cn } from '../../lib/utils';
import type { CustomEmoji } from '@enzyme/api-client';

interface MessageActionBarProps {
  reactionPickerOpen: boolean;
  onReactionPickerOpenChange: (open: boolean) => void;
  onReactionSelect: (emoji: string) => void;
  onReplyClick?: () => void;
  onCopyLink: () => void;
  onMarkUnread: () => void;
  showDropdown: boolean;
  onDropdownChange: (open: boolean) => void;
  /** If provided, shows edit option in dropdown */
  onEdit?: () => void;
  /** If provided, shows delete option in dropdown */
  onDelete?: () => void;
  /** If provided, shows pin/unpin option in dropdown */
  onPin?: () => void;
  isPinned?: boolean;
  customEmojis?: CustomEmoji[];
}

export function MessageActionBar({
  reactionPickerOpen,
  onReactionPickerOpenChange,
  onReactionSelect,
  onReplyClick,
  onCopyLink,
  onMarkUnread,
  showDropdown,
  onDropdownChange,
  onEdit,
  onDelete,
  onPin,
  isPinned,
  customEmojis,
}: MessageActionBarProps) {
  const handleEmojiSelect = (emoji: string) => {
    onReactionSelect(emoji);
    onReactionPickerOpenChange(false);
  };

  return (
    <div
      className={cn(
        'absolute top-0 right-4 flex -translate-y-1/2 items-center rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-900',
        showDropdown && 'bg-gray-100 dark:bg-gray-800',
      )}
    >
      <DialogTrigger isOpen={reactionPickerOpen} onOpenChange={onReactionPickerOpenChange}>
        <Tooltip content="Add reaction">
          <IconButton aria-label="Add reaction" className="group/btn rounded-l-lg">
            <FaceSmileIcon className="h-4 w-4 transition-transform group-hover/btn:scale-110" />
          </IconButton>
        </Tooltip>
        <Popover placement="bottom end">
          <EmojiGrid onSelect={handleEmojiSelect} customEmojis={customEmojis} />
        </Popover>
      </DialogTrigger>

      {onReplyClick && (
        <Tooltip content="Reply in thread">
          <IconButton
            onPress={onReplyClick}
            aria-label="Reply in thread"
            className="group/btn rounded-none"
          >
            <ChatBubbleBottomCenterTextIcon className="h-4 w-4 transition-transform group-hover/btn:scale-110" />
          </IconButton>
        </Tooltip>
      )}

      <Tooltip content="More options">
        <Menu
          open={showDropdown}
          onOpenChange={onDropdownChange}
          align="end"
          trigger={
            <IconButton
              aria-label="More options"
              className={cn(
                'group/btn rounded-r-lg',
                showDropdown && 'bg-gray-100 dark:bg-gray-700',
              )}
            >
              <EllipsisVerticalIcon className="h-4 w-4 transition-transform group-hover/btn:scale-110" />
            </IconButton>
          }
        >
          <MenuItem onAction={onCopyLink} icon={<LinkIcon className="h-4 w-4" />}>
            Copy link to message
          </MenuItem>
          <MenuItem onAction={onMarkUnread} icon={<EyeSlashIcon className="h-4 w-4" />}>
            Mark unread
          </MenuItem>
          {onPin && (
            <MenuItem onAction={onPin} icon={<PinOutlineIcon className="h-4 w-4" />}>
              {isPinned ? 'Unpin message' : 'Pin message'}
            </MenuItem>
          )}
          {onEdit && (
            <MenuItem onAction={onEdit} icon={<PencilSquareIcon className="h-4 w-4" />}>
              Edit message
            </MenuItem>
          )}
          {onDelete && (
            <MenuItem onAction={onDelete} variant="danger" icon={<TrashIcon className="h-4 w-4" />}>
              Delete message
            </MenuItem>
          )}
        </Menu>
      </Tooltip>
    </div>
  );
}
