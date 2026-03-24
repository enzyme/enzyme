import { createNavigationContainerRef } from '@react-navigation/native';
import type { MainStackParamList } from './types';

export const navigationRef = createNavigationContainerRef<MainStackParamList>();

export function navigateToChannel(workspaceId: string, channelId: string, channelName: string) {
  if (navigationRef.isReady()) {
    navigationRef.navigate('Channel', { workspaceId, channelId, channelName });
  }
}

export function navigateToThread(workspaceId: string, channelId: string, parentMessageId: string) {
  if (navigationRef.isReady()) {
    navigationRef.navigate('Thread', { workspaceId, channelId, parentMessageId });
  }
}
