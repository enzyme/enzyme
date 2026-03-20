import { useCallback, useRef } from 'react';
import { workspacesApi } from '@enzyme/api-client';

const TYPING_THROTTLE = 2000; // Don't re-send typing.start more than once per 2 seconds
const TYPING_STOP_DELAY = 3000; // 3 seconds of no typing before auto-sending stop

export function useTyping(workspaceId: string, channelId: string) {
  const isTypingRef = useRef(false);
  const lastSentRef = useRef(0);
  const stopTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const sendTypingStart = useCallback(async () => {
    try {
      await workspacesApi.startTyping(workspaceId, channelId);
    } catch {
      // Ignore errors
    }
  }, [workspaceId, channelId]);

  const sendTypingStop = useCallback(async () => {
    try {
      await workspacesApi.stopTyping(workspaceId, channelId);
    } catch {
      // Ignore errors
    }
  }, [workspaceId, channelId]);

  const onTyping = useCallback(() => {
    // Clear existing stop timeout
    if (stopTimeoutRef.current) {
      clearTimeout(stopTimeoutRef.current);
    }

    const now = Date.now();

    // Send typing.start immediately, but throttle to avoid spamming
    if (!isTypingRef.current || now - lastSentRef.current > TYPING_THROTTLE) {
      isTypingRef.current = true;
      lastSentRef.current = now;
      sendTypingStart();
    }

    // Set up auto-stop
    stopTimeoutRef.current = setTimeout(() => {
      if (isTypingRef.current) {
        isTypingRef.current = false;
        sendTypingStop();
      }
    }, TYPING_STOP_DELAY);
  }, [sendTypingStart, sendTypingStop]);

  const onStopTyping = useCallback(() => {
    if (stopTimeoutRef.current) {
      clearTimeout(stopTimeoutRef.current);
    }

    if (isTypingRef.current) {
      isTypingRef.current = false;
      sendTypingStop();
    }
  }, [sendTypingStop]);

  return {
    onTyping,
    onStopTyping,
  };
}
