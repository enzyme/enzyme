import { Button as AriaButton } from "react-aria-components";
import {
  FaceSmileIcon,
  ChatBubbleBottomCenterTextIcon,
  EllipsisVerticalIcon,
  LinkIcon,
  EyeSlashIcon,
  PencilSquareIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import { Tooltip, Menu, MenuItem } from "../ui";
import { ReactionPicker } from "./ReactionPicker";
import { cn } from "../../lib/utils";

interface MessageActionBarProps {
  showReactionPicker: boolean;
  onReactionPickerToggle: () => void;
  onReactionSelect: (emoji: string) => void;
  onReplyClick: () => void;
  onCopyLink: () => void;
  onMarkUnread: () => void;
  showDropdown: boolean;
  onDropdownChange: (open: boolean) => void;
  /** If provided, shows edit option in dropdown */
  onEdit?: () => void;
  /** If provided, shows delete option in dropdown */
  onDelete?: () => void;
}

export function MessageActionBar({
  showReactionPicker,
  onReactionPickerToggle,
  onReactionSelect,
  onReplyClick,
  onCopyLink,
  onMarkUnread,
  showDropdown,
  onDropdownChange,
  onEdit,
  onDelete,
}: MessageActionBarProps) {
  return (
    <>
      <div
        className={cn(
          "absolute right-4 top-0 -translate-y-1/2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm flex items-center",
          showDropdown && "bg-gray-50 dark:bg-gray-800/50",
        )}
      >
        <Tooltip content="Add reaction">
          <AriaButton
            onPress={onReactionPickerToggle}
            className="group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-l-lg"
          >
            <FaceSmileIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
          </AriaButton>
        </Tooltip>

        <Tooltip content="Reply in thread">
          <AriaButton
            onPress={onReplyClick}
            className="group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <ChatBubbleBottomCenterTextIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
          </AriaButton>
        </Tooltip>

        <Tooltip content="More options">
          <Menu
            open={showDropdown}
            onOpenChange={onDropdownChange}
            align="end"
            trigger={
              <AriaButton
                className={cn(
                  "group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-r-lg",
                  showDropdown && "bg-gray-100 dark:bg-gray-700",
                )}
                aria-label="More options"
              >
                <EllipsisVerticalIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
              </AriaButton>
            }
          >
            <MenuItem
              onAction={onCopyLink}
              icon={<LinkIcon className="w-4 h-4" />}
            >
              Copy link to message
            </MenuItem>
            <MenuItem
              onAction={onMarkUnread}
              icon={<EyeSlashIcon className="w-4 h-4" />}
            >
              Mark unread
            </MenuItem>
            {onEdit && (
              <MenuItem
                onAction={onEdit}
                icon={<PencilSquareIcon className="w-4 h-4" />}
              >
                Edit message
              </MenuItem>
            )}
            {onDelete && (
              <MenuItem
                onAction={onDelete}
                variant="danger"
                icon={<TrashIcon className="w-4 h-4" />}
              >
                Delete message
              </MenuItem>
            )}
          </Menu>
        </Tooltip>
      </div>

      {/* Reaction picker */}
      {showReactionPicker && (
        <div className="absolute right-4 top-8 z-10">
          <ReactionPicker onSelect={onReactionSelect} />
        </div>
      )}
    </>
  );
}
