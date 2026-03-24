import { useEffect, useRef } from 'react';
import * as Notifications from 'expo-notifications';
import { navigateToChannel, navigateToThread } from '../navigation/navigationRef';
import { useAppState } from './useAppState';

const SUPPRESS = {
  shouldShowAlert: false,
  shouldPlaySound: false,
  shouldSetBadge: false,
  shouldShowBanner: false,
  shouldShowList: false,
} as const;

/** Configure foreground notification behavior, handle taps, and manage badge. */
export function useNotificationHandler(isAuthenticated: boolean): void {
  const hasHandledColdStart = useRef(false);

  // Always suppress notifications while the app is foregrounded.
  // The SSE connection delivers real-time updates already — showing a
  // system notification on top would be redundant and noisy.
  useEffect(() => {
    if (!isAuthenticated) return;

    Notifications.setNotificationHandler({
      handleNotification: async () => SUPPRESS,
    });

    return () => {
      Notifications.setNotificationHandler(null);
    };
  }, [isAuthenticated]);

  // Handle notification taps (warm start)
  useEffect(() => {
    if (!isAuthenticated) return;

    const subscription = Notifications.addNotificationResponseReceivedListener((response) => {
      handleNotificationTap(response.notification.request.content.data);
    });

    return () => subscription.remove();
  }, [isAuthenticated]);

  // Handle cold start — check if app was launched from a notification
  useEffect(() => {
    if (!isAuthenticated || hasHandledColdStart.current) return;
    hasHandledColdStart.current = true;

    Notifications.getLastNotificationResponseAsync().then((response) => {
      if (response) {
        handleNotificationTap(response.notification.request.content.data);
      }
    });
  }, [isAuthenticated]);

  // Clear badge on foreground
  useAppState({
    onForeground: () => {
      Notifications.setBadgeCountAsync(0);
    },
  });
}

function handleNotificationTap(data: Record<string, unknown> | undefined) {
  if (!data) return;

  const { workspace_id, channel_id, channel_name, thread_parent_id } = data as {
    workspace_id?: string;
    channel_id?: string;
    channel_name?: string;
    thread_parent_id?: string;
  };

  if (!workspace_id || !channel_id) return;

  if (thread_parent_id) {
    navigateToThread(workspace_id, channel_id, thread_parent_id);
  } else {
    navigateToChannel(workspace_id, channel_id, channel_name || '');
  }
}
