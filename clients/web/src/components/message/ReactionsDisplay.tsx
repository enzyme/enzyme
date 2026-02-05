import { Button as AriaButton } from "react-aria-components";
import { Tooltip } from "../ui";
import { cn } from "../../lib/utils";

export interface ReactionGroup {
  emoji: string;
  count: number;
  userIds: string[];
  hasOwn: boolean;
}

interface ReactionsDisplayProps {
  reactions: ReactionGroup[];
  memberNames: Record<string, string>;
  onReactionClick: (emoji: string, hasOwn: boolean) => void;
}

export function ReactionsDisplay({
  reactions,
  memberNames,
  onReactionClick,
}: ReactionsDisplayProps) {
  if (reactions.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-1 mt-1">
      {reactions.map(({ emoji, count, userIds, hasOwn }) => {
        const userNames = userIds
          .map((id) => memberNames[id] || "Unknown")
          .join(", ");
        return (
          <Tooltip key={emoji} content={userNames}>
            <AriaButton
              onPress={() => onReactionClick(emoji, hasOwn)}
              className={cn(
                "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-sm border transition-colors",
                hasOwn
                  ? "bg-primary-100 dark:bg-primary-900/30 border-primary-300 dark:border-primary-700"
                  : "bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600",
              )}
            >
              <span>{emoji}</span>
              <span className="text-xs text-gray-600 dark:text-gray-300">
                {count}
              </span>
            </AriaButton>
          </Tooltip>
        );
      })}
    </div>
  );
}

/** Helper to group reactions by emoji */
export function groupReactionsByEmoji(
  reactions: Array<{ emoji: string; user_id: string }> | undefined,
  currentUserId: string | undefined,
): ReactionGroup[] {
  const groups = (reactions || []).reduce(
    (acc, reaction) => {
      if (!acc[reaction.emoji]) {
        acc[reaction.emoji] = {
          emoji: reaction.emoji,
          count: 0,
          userIds: [],
          hasOwn: false,
        };
      }
      acc[reaction.emoji].count++;
      acc[reaction.emoji].userIds.push(reaction.user_id);
      if (reaction.user_id === currentUserId) {
        acc[reaction.emoji].hasOwn = true;
      }
      return acc;
    },
    {} as Record<string, ReactionGroup>,
  );

  return Object.values(groups);
}

/** Helper to create member names lookup */
export function createMemberNamesMap(
  members: Array<{ user_id: string; display_name: string }> | undefined,
): Record<string, string> {
  return (members || []).reduce(
    (acc, member) => {
      acc[member.user_id] = member.display_name;
      return acc;
    },
    {} as Record<string, string>,
  );
}
