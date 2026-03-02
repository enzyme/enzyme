import { useEffect, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { useAuth } from './useAuth';
import { useChannels } from './useChannels';

const APP_NAME = 'Enzyme';

/**
 * Sets document.title based on the current page context.
 *
 * Reads :workspaceId from the URL automatically via useParams.
 * Within a workspace route: "(N) Page Title - Workspace Name"
 *   where N is the total notification_count (mentions + DMs), not total unreads.
 * Outside a workspace route: "Page Title - Enzyme"
 * Fallback: "Enzyme"
 */
export function usePageTitle(pageTitle?: string) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { workspaces } = useAuth();
  const { data: channelsData } = useChannels(workspaceId);

  const workspaceName = useMemo(
    () => workspaces?.find((ws) => ws.id === workspaceId)?.name,
    [workspaces, workspaceId],
  );

  const totalNotifications = useMemo(
    () =>
      workspaceId
        ? (channelsData?.channels.reduce((sum, c) => sum + c.notification_count, 0) ?? 0)
        : 0,
    [workspaceId, channelsData?.channels],
  );

  useEffect(() => {
    const parts: string[] = [];

    if (workspaceId) {
      const notifPrefix = totalNotifications > 0 ? `(${totalNotifications}) ` : '';
      if (pageTitle) parts.push(pageTitle);
      if (workspaceName) parts.push(workspaceName);
      document.title = parts.length > 0 ? notifPrefix + parts.join(' - ') : APP_NAME;
    } else {
      if (pageTitle) parts.push(pageTitle);
      parts.push(APP_NAME);
      document.title = parts.join(' - ');
    }
  }, [pageTitle, workspaceId, workspaceName, totalNotifications]);
}
