import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Button as AriaButton } from 'react-aria-components';
import {
  FaceSmileIcon,
  ChatBubbleBottomCenterTextIcon,
  EllipsisVerticalIcon,
  LinkIcon,
  EyeSlashIcon,
} from '@heroicons/react/24/outline';
import { Avatar, Tooltip, Menu, MenuItem, toast } from '../ui';
import { ReactionPicker } from './ReactionPicker';
import { ThreadRepliesIndicator } from './ThreadRepliesIndicator';
import { useAuth, useAddReaction, useRemoveReaction, useWorkspaceMembers } from '../../hooks';
import { useMarkMessageUnread } from '../../hooks/useMessages';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { cn, formatTime } from '../../lib/utils';
import type { MessageWithUser } from '@feather/api-client';

interface SystemMessageProps {
  message: MessageWithUser;
  channelId: string;
}

export function SystemMessage({ message, channelId }: SystemMessageProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const [showActions, setShowActions] = useState(false);
  const [showReactionPicker, setShowReactionPicker] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const { user } = useAuth();
  const { openThread } = useThreadPanel();
  const { openProfile } = useProfilePanel();
  const addReaction = useAddReaction(channelId);
  const removeReaction = useRemoveReaction(channelId);
  const markUnread = useMarkMessageUnread(workspaceId || '');
  const { data: membersData } = useWorkspaceMembers(workspaceId);

  const handleCopyLink = () => {
    const url = `${window.location.origin}/w/${workspaceId}/c/${channelId}?msg=${message.id}`;
    navigator.clipboard.writeText(url);
    toast('Link copied to clipboard', 'success');
    setShowDropdown(false);
  };

  // Create a lookup map from user ID to display name
  const memberNames = (membersData?.members || []).reduce((acc, member) => {
    acc[member.user_id] = member.display_name;
    return acc;
  }, {} as Record<string, string>);

  // Group reactions by emoji
  const reactionGroups = (message.reactions || []).reduce((acc, reaction) => {
    if (!acc[reaction.emoji]) {
      acc[reaction.emoji] = { emoji: reaction.emoji, count: 0, userIds: [], hasOwn: false };
    }
    acc[reaction.emoji].count++;
    acc[reaction.emoji].userIds.push(reaction.user_id);
    if (reaction.user_id === user?.id) {
      acc[reaction.emoji].hasOwn = true;
    }
    return acc;
  }, {} as Record<string, { emoji: string; count: number; userIds: string[]; hasOwn: boolean }>);

  const handleReactionClick = (emoji: string, hasOwn: boolean) => {
    if (hasOwn) {
      removeReaction.mutate({ messageId: message.id, emoji });
    } else {
      addReaction.mutate({ messageId: message.id, emoji });
    }
  };

  const handleAddReaction = (emoji: string) => {
    addReaction.mutate({ messageId: message.id, emoji });
    setShowReactionPicker(false);
  };

  // Build the system message content
  const systemEvent = message.system_event;
  let contentText = message.content;

  if (systemEvent) {
    switch (systemEvent.event_type) {
      case 'user_joined':
        contentText = `joined #${systemEvent.channel_name}`;
        break;
      case 'user_left':
        contentText = `left #${systemEvent.channel_name}`;
        break;
      case 'user_added':
        if (systemEvent.actor_display_name) {
          contentText = `was added by ${systemEvent.actor_display_name}`;
        } else {
          contentText = `was added to #${systemEvent.channel_name}`;
        }
        break;
    }
  }

  return (
    <div
      className={cn(
        "group px-4 py-1.5 relative",
        "hover:bg-gray-50 dark:hover:bg-gray-800/50",
        showDropdown && "bg-gray-50 dark:bg-gray-800/50"
      )}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => {
        if (!showDropdown) {
          setShowActions(false);
        }
        setShowReactionPicker(false);
      }}
    >
      <div className="flex items-start gap-3">
        {/* Avatar - uses the user who joined/left/was added */}
        <Avatar
          src={message.user_avatar_url}
          name={message.user_display_name || systemEvent?.user_display_name || 'System'}
          id={message.user_id}
          size="md"
          onClick={message.user_id ? () => openProfile(message.user_id!) : undefined}
        />

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex items-baseline gap-2">
            <button
              type="button"
              onClick={message.user_id ? () => openProfile(message.user_id!) : undefined}
              className={cn(
                "font-medium text-gray-900 dark:text-white",
                message.user_id && "hover:underline cursor-pointer"
              )}
            >
              {message.user_display_name || systemEvent?.user_display_name || 'System'}
            </button>
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
          </div>

          {/* System message content - styled differently */}
          <div className="text-gray-600 dark:text-gray-400 text-sm italic">
            {contentText}
          </div>

          {/* Reactions */}
          {Object.values(reactionGroups).length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {Object.values(reactionGroups).map(({ emoji, count, userIds, hasOwn }) => {
                const userNamesStr = userIds
                  .map((id) => memberNames[id] || 'Unknown')
                  .join(', ');
                return (
                  <Tooltip key={emoji} content={userNamesStr}>
                    <AriaButton
                      onPress={() => handleReactionClick(emoji, hasOwn)}
                      className={cn(
                        'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-sm border transition-colors',
                        hasOwn
                          ? 'bg-primary-100 dark:bg-primary-900/30 border-primary-300 dark:border-primary-700'
                          : 'bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 hover:bg-gray-200 dark:hover:bg-gray-600'
                      )}
                    >
                      <span>{emoji}</span>
                      <span className="text-xs text-gray-600 dark:text-gray-300">{count}</span>
                    </AriaButton>
                  </Tooltip>
                );
              })}
            </div>
          )}

          {/* Thread replies indicator */}
          <ThreadRepliesIndicator
            messageId={message.id}
            replyCount={message.reply_count}
            lastReplyAt={message.last_reply_at}
            threadParticipants={message.thread_participants}
          />
        </div>
      </div>

      {/* Action buttons */}
      {showActions && (
        <div className={cn(
          "absolute right-4 top-0 -translate-y-1/2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm flex items-center",
          showDropdown && "bg-gray-50 dark:bg-gray-800/50"
        )}>
          <Tooltip content="Add reaction">
            <AriaButton
              onPress={() => setShowReactionPicker(!showReactionPicker)}
              className="group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-l-lg"
            >
              <FaceSmileIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
            </AriaButton>
          </Tooltip>

          <Tooltip content="Reply in thread">
            <AriaButton
              onPress={() => openThread(message.id)}
              className="group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <ChatBubbleBottomCenterTextIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
            </AriaButton>
          </Tooltip>

          <Tooltip content="More options">
            <Menu
              open={showDropdown}
              onOpenChange={setShowDropdown}
              align="end"
              trigger={
                <AriaButton
                  className={cn(
                    'group/btn p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-r-lg',
                    showDropdown && 'bg-gray-100 dark:bg-gray-700'
                  )}
                  aria-label="More options"
                >
                  <EllipsisVerticalIcon className="w-4 h-4 text-gray-500 transition-transform group-hover/btn:scale-110 group-hover/btn:text-gray-700 dark:group-hover/btn:text-gray-300" />
                </AriaButton>
              }
            >
              <MenuItem
                onAction={handleCopyLink}
                icon={<LinkIcon className="w-4 h-4" />}
              >
                Copy link to message
              </MenuItem>
              <MenuItem
                onAction={() => markUnread.mutate(message.id)}
                icon={<EyeSlashIcon className="w-4 h-4" />}
              >
                Mark unread
              </MenuItem>
            </Menu>
          </Tooltip>
        </div>
      )}

      {/* Reaction picker */}
      {showReactionPicker && (
        <div className="absolute right-4 top-8 z-10">
          <ReactionPicker onSelect={handleAddReaction} />
        </div>
      )}
    </div>
  );
}
