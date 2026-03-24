import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';
import { authApi, getAuthToken } from '@enzyme/api-client';

const DEVICE_ID_KEY = 'enzyme_device_id';

let registeredTokenId: string | null = null;

function generateUUID(): string {
  const bytes = new Uint8Array(16);
  for (let i = 0; i < 16; i++) bytes[i] = Math.floor(Math.random() * 256);
  bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
  bytes[8] = (bytes[8] & 0x3f) | 0x80; // variant 1
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

async function getDeviceId(): Promise<string> {
  const existing = await SecureStore.getItemAsync(DEVICE_ID_KEY);
  if (existing) return existing;

  const id = generateUUID();
  await SecureStore.setItemAsync(DEVICE_ID_KEY, id);
  return id;
}

/** Request notification permissions. Returns true if granted. */
export async function requestPermissions(): Promise<boolean> {
  const { status: existing } = await Notifications.getPermissionsAsync();
  if (existing === 'granted') return true;

  const { status } = await Notifications.requestPermissionsAsync({
    ios: { allowAlert: true, allowBadge: true, allowSound: true },
  });
  return status === 'granted';
}

/** Get the native push token (FCM or APNs). Returns null on simulator. */
export async function getDevicePushToken(): Promise<string | null> {
  if (!Device.isDevice) return null;

  try {
    const token = await Notifications.getDevicePushTokenAsync();
    return token.data as string;
  } catch {
    return null;
  }
}

/** Register device token with the Enzyme backend. Idempotent (backend upserts). */
export async function registerPushToken(): Promise<void> {
  try {
    if (!getAuthToken()) return;

    const token = await getDevicePushToken();
    if (!token) return;

    const platform = Platform.OS === 'ios' ? 'apns' : 'fcm';
    const deviceId = await getDeviceId();

    const response = await authApi.registerDeviceToken({
      token,
      platform,
      device_id: deviceId,
    });
    registeredTokenId = response.id;
  } catch (err) {
    console.warn('Push token registration failed:', err);
  }
}

/** Unregister the current device token from the backend. Call on logout. */
export async function unregisterPushToken(): Promise<void> {
  if (!registeredTokenId) return;

  try {
    await authApi.unregisterDeviceToken(registeredTokenId);
  } catch (err) {
    console.warn('Push token unregistration failed:', err);
  } finally {
    registeredTokenId = null;
  }
}

/** Set up the token refresh listener. Returns cleanup function. */
export function onTokenRefresh(callback: () => void): () => void {
  const subscription = Notifications.addPushTokenListener(() => {
    callback();
  });
  return () => subscription.remove();
}
