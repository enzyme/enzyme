import { useCallback } from 'react';
import { useParams, useSearchParams, useLocation } from 'react-router-dom';
import { useIsMobile } from './useIsMobile';

export type MobilePanel = 'switcher' | 'sidebar' | 'channel' | 'thread' | 'profile';

// Routes that render into <Outlet /> and should show as main content.
// Update this when adding new workspace-level pages.
const CONTENT_PAGE_RE = /\/workspaces\/[^/]+\/(unreads|threads|scheduled)$/;

export function useMobileNav() {
  const { channelId } = useParams<{ channelId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const location = useLocation();
  const isMobile = useIsMobile();

  const hasThread = searchParams.has('thread');
  const hasProfile = searchParams.has('profile');
  const hasSwitcher = searchParams.has('switcher');

  const isContentPage = CONTENT_PAGE_RE.test(location.pathname);

  let activePanel: MobilePanel;
  if (hasSwitcher) {
    activePanel = 'switcher';
  } else if (hasThread) {
    activePanel = 'thread';
  } else if (hasProfile) {
    activePanel = 'profile';
  } else if (channelId || isContentPage) {
    activePanel = 'channel';
  } else {
    activePanel = 'sidebar';
  }

  const openSwitcher = useCallback(() => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set('switcher', '');
        next.delete('thread');
        next.delete('profile');
        return next;
      },
      { replace: !isMobile },
    );
  }, [setSearchParams, isMobile]);

  return { activePanel, openSwitcher };
}
