import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { SSEConnection } from '../lib/sse';
import {
  handleNewMessage,
  handleMessageUpdated,
  handleMessageDeleted,
  handleReactionAdded,
  handleReactionRemoved,
  handleChannelCreated,
  handleChannelsInvalidate,
  handleChannelUpdated,
  handleChannelArchived,
  handleMemberAdded,
  handleMemberRemoved,
  handleChannelRead,
  handleEmojiCreated,
  handleEmojiDeleted,
  handleWorkspaceUpdated,
  handleScheduledMessageChange,
  handleScheduledMessageSent,
  handleMessagePinned,
  handleMessageUnpinned,
  handleMemberBanned,
  handleMemberUnbanned,
  handleMemberLeft,
  handleMemberRoleChanged,
  handleTypingStart,
  handleTypingStop,
  handlePresenceChanged,
  handlePresenceInitial,
  handleNotification,
  clearPresence,
  authKeys,
} from '@enzyme/shared';

export function useSSE(workspaceId: string | undefined) {
  const [isConnected, setIsConnected] = useState(false);
  const [hasBeenConnected, setHasBeenConnected] = useState(false);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!workspaceId) {
      return;
    }

    const connection = new SSEConnection(workspaceId);

    // Handle disconnect
    connection.setOnDisconnect(() => {
      setIsConnected(false);
    });

    // Handle 403 — stop reconnecting and refresh auth state
    connection.setOnForbidden(() => {
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    });

    // Handle connected event
    connection.on('connected', () => {
      setHasBeenConnected(true);
      setIsConnected(true);
    });

    // --- Message events ---
    connection.on('message.new', (event) => {
      handleNewMessage(queryClient, workspaceId, event.data);
    });

    connection.on('message.updated', (event) => {
      handleMessageUpdated(queryClient, event.data);
    });

    connection.on('message.deleted', (event) => {
      handleMessageDeleted(queryClient, event.data);
    });

    // --- Reaction events ---
    connection.on('reaction.added', (event) => {
      handleReactionAdded(queryClient, event.data);
    });

    connection.on('reaction.removed', (event) => {
      handleReactionRemoved(queryClient, event.data);
    });

    // --- Channel events ---
    connection.on('channel.created', (event) => {
      handleChannelCreated(queryClient, workspaceId, event.data);
    });

    connection.on('channels.invalidate', () => {
      handleChannelsInvalidate(queryClient, workspaceId);
    });

    connection.on('channel.updated', (event) => {
      handleChannelUpdated(queryClient, workspaceId, event.data);
    });

    connection.on('channel.archived', (event) => {
      handleChannelArchived(queryClient, workspaceId, event.data);
    });

    connection.on('channel.member_added', (event) => {
      handleMemberAdded(queryClient, workspaceId, event.data);
    });

    connection.on('channel.member_removed', (event) => {
      handleMemberRemoved(queryClient, workspaceId, event.data);
    });

    connection.on('channel.read', (event) => {
      handleChannelRead(queryClient, workspaceId, event.data);
    });

    // --- Emoji events ---
    connection.on('emoji.created', (event) => {
      handleEmojiCreated(queryClient, workspaceId, event.data);
    });

    connection.on('emoji.deleted', (event) => {
      handleEmojiDeleted(queryClient, workspaceId, event.data);
    });

    // --- Workspace events ---
    connection.on('workspace.updated', () => {
      handleWorkspaceUpdated(queryClient, workspaceId);
    });

    // --- Scheduled message events ---
    connection.on('scheduled_message.created', () => {
      handleScheduledMessageChange(queryClient, workspaceId);
    });

    connection.on('scheduled_message.updated', () => {
      handleScheduledMessageChange(queryClient, workspaceId);
    });

    connection.on('scheduled_message.deleted', () => {
      handleScheduledMessageChange(queryClient, workspaceId);
    });

    connection.on('scheduled_message.sent', (event) => {
      handleScheduledMessageSent(queryClient, workspaceId, event.data);
    });

    connection.on('scheduled_message.failed', () => {
      handleScheduledMessageChange(queryClient, workspaceId);
    });

    // --- Pin events ---
    connection.on('message.pinned', (event) => {
      handleMessagePinned(queryClient, event.data);
    });

    connection.on('message.unpinned', (event) => {
      handleMessageUnpinned(queryClient, event.data);
    });

    // --- Member events ---
    connection.on('member.banned', (event) => {
      handleMemberBanned(queryClient, workspaceId, event.data);
    });

    connection.on('member.unbanned', (event) => {
      handleMemberUnbanned(queryClient, workspaceId, event.data);
    });

    connection.on('member.left', () => {
      handleMemberLeft(queryClient, workspaceId);
    });

    connection.on('member.role_changed', (event) => {
      handleMemberRoleChanged(queryClient, workspaceId, event.data);
    });

    // --- Typing & Presence events ---
    connection.on('typing.start', (event) => {
      handleTypingStart(event.data);
    });

    connection.on('typing.stop', (event) => {
      handleTypingStop(event.data);
    });

    connection.on('presence.changed', (event) => {
      handlePresenceChanged(event.data);
    });

    connection.on('presence.initial', (event) => {
      handlePresenceInitial(event.data);
    });

    // --- Notification events ---
    connection.on('notification', (event) => {
      handleNotification(queryClient, workspaceId, event.data);
      // Push notifications for mobile will be handled separately (#207)
    });

    connection.connect();

    return () => {
      connection.disconnect();
      clearPresence();
      setHasBeenConnected(false);
      setIsConnected(false);
    };
  }, [workspaceId, queryClient]);

  const isReconnecting = hasBeenConnected && !isConnected;

  return { isReconnecting };
}
