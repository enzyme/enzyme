import { useEffect, useRef } from 'react';
import { AppState, type AppStateStatus } from 'react-native';

export function useAppState(callbacks: {
  onForeground?: () => void;
  onBackground?: () => void;
}): void {
  const callbacksRef = useRef(callbacks);
  callbacksRef.current = callbacks;

  useEffect(() => {
    let previous = AppState.currentState;

    const subscription = AppState.addEventListener('change', (next: AppStateStatus) => {
      const wasBackground = previous === 'background' || previous === 'inactive';
      const isActive = next === 'active';
      const wasActive = previous === 'active';
      const isBackground = next === 'background' || next === 'inactive';

      if (wasBackground && isActive) {
        callbacksRef.current.onForeground?.();
      } else if (wasActive && isBackground) {
        callbacksRef.current.onBackground?.();
      }

      previous = next;
    });

    return () => subscription.remove();
  }, []);
}
