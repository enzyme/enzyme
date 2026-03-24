import { useEffect, useRef } from 'react';
import { requestPermissions, registerPushToken, onTokenRefresh } from '../lib/notifications';

/** Manage push notification token lifecycle tied to authentication state. */
export function usePushNotifications(isAuthenticated: boolean): void {
  const wasAuthenticated = useRef(false);

  useEffect(() => {
    if (isAuthenticated && !wasAuthenticated.current) {
      // User just logged in — request permissions and register token
      wasAuthenticated.current = true;

      (async () => {
        const granted = await requestPermissions();
        if (granted) {
          await registerPushToken();
        }
      })();
    } else if (!isAuthenticated && wasAuthenticated.current) {
      // User just logged out — cleanup handled by logout button
      wasAuthenticated.current = false;
    }
  }, [isAuthenticated]);

  // Listen for token refreshes while authenticated
  useEffect(() => {
    if (!isAuthenticated) return;

    return onTokenRefresh(() => {
      registerPushToken();
    });
  }, [isAuthenticated]);
}
