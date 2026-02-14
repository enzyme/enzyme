import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Avatar, toast } from '../ui';
import { ThreadRepliesIndicator } from './ThreadRepliesIndicator';
import { MessageActionBar } from './MessageActionBar';
import { ReactionsDisplay } from './ReactionsDisplay';
import { groupReactionsByEmoji, createMemberNamesMap } from './reactionUtils';
import { useAuth, useAddReaction, useRemoveReaction, useWorkspaceMembers } from '../../hooks';
import { useMarkMessageUnread } from '../../hooks/useMessages';
import { useCustomEmojiMap, useCustomEmojis } from '../../hooks/useCustomEmojis';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { cn, formatTime } from '../../lib/utils';
import type { MessageWithUser } from '@feather/api-client';

interface SystemMessageProps {
  message: MessageWithUser;
  channelId: string;
}

export function SystemMessage({ message, channelId }: SystemMessageProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const customEmojiMap = useCustomEmojiMap(workspaceId);
  const { data: customEmojis } = useCustomEmojis(workspaceId);
  const [showActions, setShowActions] = useState(false);
  const [reactionPickerOpen, setReactionPickerOpen] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const { user } = useAuth();
  const { openThread } = useThreadPanel();
  const { openProfile } = useProfilePanel();
  const addReaction = useAddReaction(channelId);
  const removeReaction = useRemoveReaction(channelId);
  const markUnread = useMarkMessageUnread(workspaceId || '');
  const { data: membersData } = useWorkspaceMembers(workspaceId);

  const memberNames = createMemberNamesMap(membersData?.members);
  const reactionGroups = groupReactionsByEmoji(message.reactions, user?.id);

  const handleCopyLink = () => {
    const url = `${window.location.origin}/workspaces/${workspaceId}/channels/${channelId}?msg=${message.id}`;
    navigator.clipboard.writeText(url);
    toast('Link copied to clipboard', 'success');
    setShowDropdown(false);
  };

  const handleReactionClick = (emoji: string, hasOwn: boolean) => {
    if (hasOwn) {
      removeReaction.mutate({ messageId: message.id, emoji });
    } else {
      addReaction.mutate({ messageId: message.id, emoji });
    }
  };

  const handleAddReaction = (emoji: string) => {
    addReaction.mutate({ messageId: message.id, emoji });
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
      case 'user_converted_channel':
        contentText = `converted this conversation to #${systemEvent.channel_name}`;
        break;
    }
  }

  return (
    <div
      className={cn(
        'group relative px-4 py-1.5',
        'hover:bg-gray-50 dark:hover:bg-gray-800/50',
        showDropdown && 'bg-gray-50 dark:bg-gray-800/50',
      )}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => {
        if (!showDropdown && !reactionPickerOpen) {
          setShowActions(false);
        }
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
        <div className="min-w-0 flex-1">
          {/* Header */}
          <div className="flex items-baseline gap-2">
            <button
              type="button"
              onClick={message.user_id ? () => openProfile(message.user_id!) : undefined}
              className={cn(
                'font-medium text-gray-900 dark:text-white',
                message.user_id && 'cursor-pointer hover:underline',
              )}
            >
              {message.user_display_name || systemEvent?.user_display_name || 'System'}
            </button>
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTime(message.created_at)}
            </span>
          </div>

          {/* System message content - styled differently */}
          <div className="text-sm italic text-gray-600 dark:text-gray-400">{contentText}</div>

          {/* Reactions */}
          <ReactionsDisplay
            reactions={reactionGroups}
            memberNames={memberNames}
            onReactionClick={handleReactionClick}
            customEmojiMap={customEmojiMap}
          />

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
        <MessageActionBar
          reactionPickerOpen={reactionPickerOpen}
          onReactionPickerOpenChange={setReactionPickerOpen}
          onReactionSelect={handleAddReaction}
          onReplyClick={() => openThread(message.id)}
          onCopyLink={handleCopyLink}
          onMarkUnread={() => markUnread.mutate(message.id)}
          showDropdown={showDropdown}
          onDropdownChange={setShowDropdown}
          customEmojis={customEmojis}
        />
      )}
    </div>
  );
}
