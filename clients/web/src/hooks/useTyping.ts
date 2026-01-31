import { useCallback, useRef } from 'react';
import { workspacesApi } from '../api/workspaces';

const TYPING_DEBOUNCE = 1000; // 1 second debounce for sending start
const TYPING_STOP_DELAY = 3000; // 3 seconds before auto-sending stop

export function useTyping(workspaceId: string, channelId: string) {
  const isTypingRef = useRef(false);
  const debounceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
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

    // If not currently typing, send start after debounce
    if (!isTypingRef.current) {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }

      debounceTimeoutRef.current = setTimeout(() => {
        isTypingRef.current = true;
        sendTypingStart();
      }, TYPING_DEBOUNCE);
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
    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current);
    }
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
