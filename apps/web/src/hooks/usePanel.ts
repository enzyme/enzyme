import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useIsMobile } from './useIsMobile';

/**
 * URL-based thread panel state
 * Uses ?thread= search param as source of truth
 */
export function useThreadPanel() {
  const [searchParams, setSearchParams] = useSearchParams();
  const isMobile = useIsMobile();
  const threadId = searchParams.get('thread');

  const openThread = useCallback(
    (id: string) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set('thread', id);
          next.delete('profile'); // Close profile when opening thread
          return next;
        },
        { replace: !isMobile },
      );
    },
    [setSearchParams, isMobile],
  );

  const closeThread = useCallback(() => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.delete('thread');
        return next;
      },
      { replace: !isMobile },
    );
  }, [setSearchParams, isMobile]);

  return { threadId, openThread, closeThread };
}

/**
 * URL-based profile panel state
 * Uses ?profile= search param as source of truth
 */
export function useProfilePanel() {
  const [searchParams, setSearchParams] = useSearchParams();
  const isMobile = useIsMobile();
  const profileUserId = searchParams.get('profile');

  const openProfile = useCallback(
    (userId: string) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set('profile', userId);
          next.delete('thread'); // Close thread when opening profile
          return next;
        },
        { replace: !isMobile },
      );
    },
    [setSearchParams, isMobile],
  );

  const closeProfile = useCallback(() => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.delete('profile');
        return next;
      },
      { replace: !isMobile },
    );
  }, [setSearchParams, isMobile]);

  return { profileUserId, openProfile, closeProfile };
}
