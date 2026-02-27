import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Avatar, toast } from '../ui';
import { ThreadRepliesIndicator } from './ThreadRepliesIndicator';
import { MessageActionBar } from './MessageActionBar';
import { ReactionsDisplay } from './ReactionsDisplay';
import { MessageContent } from './MessageContent';
import { groupReactionsByEmoji, createMemberNamesMap } from './reactionUtils';
import { useAuth, useAddReaction, useRemoveReaction, useWorkspaceMembers } from '../../hooks';
import { useMessage, useMarkMessageUnread } from '../../hooks/useMessages';
import { useCustomEmojiMap, useCustomEmojis } from '../../hooks/useCustomEmojis';
import { useChannels } from '../../hooks/useChannels';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { cn, formatTime } from '../../lib/utils';
import type { MessageWithUser } from '@enzyme/api-client';

interface SystemMessageProps {
  message: MessageWithUser;
  channelId: string;
}

export function SystemMessage({ message, channelId }: SystemMessageProps) {
  const navigate = useNavigate();
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
  const { data: channels } = useChannels(workspaceId);

  // Fetch the referenced message for pin/unpin events
  const systemEvent = message.system_event;
  const referencedMessageId =
    systemEvent?.event_type === 'message_pinned' || systemEvent?.event_type === 'message_unpinned'
      ? systemEvent.message_id
      : undefined;
  const { data: referencedMessage } = useMessage(referencedMessageId);

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
      case 'channel_renamed':
        contentText = `renamed the channel from #${systemEvent.old_channel_name} to #${systemEvent.channel_name}`;
        break;
      case 'channel_visibility_changed':
        contentText = `made the channel ${systemEvent.channel_type}`;
        break;
      case 'channel_description_updated':
        contentText = `updated the channel description`;
        break;
      case 'message_pinned':
        contentText = 'pinned a message to this channel';
        break;
      case 'message_unpinned':
        contentText = 'unpinned a message from this channel';
        break;
    }
  }

  return (
    <div
      className={cn(
        'group relative px-4 py-1.5',
        'hover:bg-gray-100 dark:hover:bg-gray-800',
        showDropdown && 'bg-gray-100 dark:bg-gray-800',
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
          gravatarSrc={message.user_gravatar_url}
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
          <div className="text-sm text-gray-600 italic dark:text-gray-400">{contentText}</div>

          {/* Inline preview of pinned/unpinned message */}
          {referencedMessage?.message && (
            <button
              type="button"
              onClick={() =>
                navigate(
                  `/workspaces/${workspaceId}/channels/${channelId}?msg=${referencedMessage.message.id}`,
                )
              }
              className="mt-1.5 block max-w-lg cursor-pointer overflow-hidden rounded-lg border border-gray-200 bg-white text-left dark:border-gray-700 dark:bg-gray-900"
            >
              <div className="flex">
                <div className="w-1 flex-shrink-0 bg-blue-500" />
                <div className="min-w-0 flex-1 px-3 py-2">
                  <div className="flex items-center gap-1.5">
                    <Avatar
                      src={referencedMessage.message.user_avatar_url}
                      gravatarSrc={referencedMessage.message.user_gravatar_url}
                      name={referencedMessage.message.user_display_name || 'Unknown'}
                      id={referencedMessage.message.user_id}
                      size="xs"
                    />
                    <span className="truncate text-sm font-medium text-gray-900 dark:text-white">
                      {referencedMessage.message.user_display_name || 'Unknown'}
                    </span>
                  </div>
                  {referencedMessage.message.content && (
                    <div className="mt-1 line-clamp-3 text-sm break-words text-gray-700 dark:text-gray-300">
                      <MessageContent
                        content={referencedMessage.message.content}
                        members={membersData?.members}
                        channels={channels?.channels}
                        customEmojiMap={customEmojiMap}
                      />
                    </div>
                  )}
                </div>
              </div>
            </button>
          )}

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
