import { createNavigationContainerRef, CommonActions } from '@react-navigation/native';
import type { MainStackParamList } from './types';

export const navigationRef = createNavigationContainerRef<MainStackParamList>();

// Only the most recent pending navigation is kept. This is intentional —
// only the last notification tap should be acted on.
let pendingNavigation: (() => void) | null = null;

function deferredNavigate(fn: () => void) {
  if (navigationRef.isReady()) {
    fn();
  } else {
    pendingNavigation = fn;
  }
}

type ScreenRoute<K extends keyof MainStackParamList = keyof MainStackParamList> = {
  [P in K]: { name: P; params: MainStackParamList[P] };
}[K];

function resetToScreen(workspaceId: string, screen: ScreenRoute) {
  deferredNavigate(() => {
    // Reset the stack so ChannelList mounts (establishing SSE connection)
    // and the target screen is pushed on top.
    navigationRef.dispatch(
      CommonActions.reset({
        index: 2,
        routes: [
          { name: 'WorkspaceSwitcher' },
          { name: 'ChannelList', params: { workspaceId } },
          screen,
        ],
      }),
    );
  });
}

export function navigateToChannel(workspaceId: string, channelId: string, channelName: string) {
  resetToScreen(workspaceId, { name: 'Channel', params: { workspaceId, channelId, channelName } });
}

export function navigateToThread(workspaceId: string, channelId: string, parentMessageId: string) {
  resetToScreen(workspaceId, {
    name: 'Thread',
    params: { workspaceId, channelId, parentMessageId },
  });
}

/** Flush any pending navigation queued before the navigator was ready. */
export function flushPendingNavigation() {
  if (pendingNavigation && navigationRef.isReady()) {
    pendingNavigation();
    pendingNavigation = null;
  }
}
