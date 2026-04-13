import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { SSEConnection } from '../lib/sse';
import {
  playNotificationSound,
  showBrowserNotification,
  unlockAudio,
  requestNotificationPermission,
} from '../lib/notificationSound';
import { toast } from '../components/ui';
import type { NotificationData } from '@enzyme/api-client';
import {
  dispatchVoiceOffer,
  dispatchVoiceICECandidate,
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
  handleVoiceJoined,
  handleVoiceLeft,
  handleVoiceSpeaking,
  handleVoiceMuted,
  authKeys,
} from '@enzyme/shared';

export function useSSE(workspaceId: string | undefined) {
  const [isConnected, setIsConnected] = useState(false);
  const [hasBeenConnected, setHasBeenConnected] = useState(false);
  const connectionRef = useRef<SSEConnection | null>(null);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  useEffect(() => {
    navigateRef.current = navigate;
  }, [navigate]);

  useEffect(() => {
    if (!workspaceId) {
      return;
    }

    const connection = new SSEConnection(workspaceId);
    connectionRef.current = connection;

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
      toast('A scheduled message failed to send', 'error');
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
      const isCurrentUser = handleMemberUnbanned(queryClient, workspaceId, event.data);
      if (isCurrentUser) {
        toast('You have been unbanned', 'success');
      }
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
      const notification = handleNotification(queryClient, workspaceId, event.data);

      // Web-specific: play sound and show browser notification when tab is not focused
      if (!document.hasFocus()) {
        playNotificationSound();

        const title = getNotificationTitle(notification);
        showBrowserNotification(title, notification.preview || '', () => {
          if (notification.channel_id && workspaceId) {
            navigateRef.current(`/workspaces/${workspaceId}/channels/${notification.channel_id}`);
          }
        });
      }
    });

    // --- Voice events ---
    connection.on('voice.joined', (event) => {
      handleVoiceJoined(event.data);
    });

    connection.on('voice.left', (event) => {
      handleVoiceLeft(event.data);
    });

    connection.on('voice.speaking', (event) => {
      handleVoiceSpeaking(event.data);
    });

    connection.on('voice.muted', (event) => {
      handleVoiceMuted(event.data);
    });

    connection.on('voice.offer', (event) => {
      dispatchVoiceOffer(event.data);
    });

    connection.on('voice.ice_candidate', (event) => {
      dispatchVoiceICECandidate(event.data);
    });

    // Unlock audio on first interaction
    const handleInteraction = () => {
      unlockAudio();
      document.removeEventListener('click', handleInteraction);
      document.removeEventListener('keydown', handleInteraction);
    };
    document.addEventListener('click', handleInteraction);
    document.addEventListener('keydown', handleInteraction);

    connection.connect();

    // Request notification permission if not yet decided (avoids prompting on every connection)
    if ('Notification' in window && Notification.permission === 'default') {
      requestNotificationPermission();
    }

    return () => {
      connection.disconnect();
      connectionRef.current = null;
      setHasBeenConnected(false);
      setIsConnected(false);
    };
  }, [workspaceId, queryClient]);

  const isReconnecting = hasBeenConnected && !isConnected;

  return { isReconnecting };
}

// Helper to format notification title
function getNotificationTitle(notification: NotificationData): string {
  const prefix = notification.type === 'dm' ? 'DM' : `#${notification.channel_name || 'channel'}`;
  const sender = notification.sender_name || 'Someone';

  switch (notification.type) {
    case 'mention':
      return `${sender} mentioned you in ${prefix}`;
    case 'dm':
      return `${sender} sent you a message`;
    case 'channel':
      return `${sender} in ${prefix} (@channel)`;
    case 'here':
      return `${sender} in ${prefix} (@here)`;
    case 'everyone':
      return `${sender} in ${prefix} (@everyone)`;
    case 'thread_reply':
      return `${sender} replied to a thread in ${prefix}`;
    default:
      return `New message from ${sender}`;
  }
}
