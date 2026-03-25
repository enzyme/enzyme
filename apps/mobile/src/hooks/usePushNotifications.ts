import { useEffect } from 'react';
import * as Notifications from 'expo-notifications';
import {
  requestPermissions,
  registerPushToken,
  registerPushTokenWithValue,
} from '../lib/notifications';

/** Manage push notification token lifecycle tied to authentication state. */
export function usePushNotifications(isAuthenticated: boolean): void {
  useEffect(() => {
    if (!isAuthenticated) return;

    (async () => {
      const granted = await requestPermissions();
      if (granted) {
        await registerPushToken();
      }
    })().catch((err) => {
      console.warn('Push notification setup failed:', err);
    });

    // Use the token provided by the listener directly to avoid calling
    // getDevicePushTokenAsync inside the callback (which re-triggers the
    // listener and can cause an infinite loop per expo docs).
    const subscription = Notifications.addPushTokenListener((devicePushToken) => {
      const token = devicePushToken.data;
      if (typeof token !== 'string') return;
      registerPushTokenWithValue(token).catch((err) => {
        console.warn('Push token refresh failed:', err);
      });
    });
    return () => subscription.remove();
  }, [isAuthenticated]);
}
