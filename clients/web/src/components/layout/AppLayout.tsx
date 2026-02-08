import { Outlet, useParams } from 'react-router-dom';
import { WorkspaceSwitcher } from '../workspace/WorkspaceSwitcher';
import { ChannelSidebar } from '../channel/ChannelSidebar';
import { ThreadPanel } from '../thread/ThreadPanel';
import { ProfilePane } from '../profile/ProfilePane';
import { useSSE } from '../../hooks';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { useSidebar } from '../../hooks/useSidebar';
import { cn } from '../../lib/utils';

export function AppLayout() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { isConnected } = useSSE(workspaceId);
  const { threadId } = useThreadPanel();
  const { profileUserId } = useProfilePanel();
  const { collapsed: sidebarCollapsed } = useSidebar();

  return (
    <div className="h-screen flex flex-col bg-white dark:bg-gray-900">
      {/* Connection Status - full width */}
      {!isConnected && workspaceId && (
        <div className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 text-sm px-4 py-1.5 text-center flex-shrink-0 border-b border-yellow-200 dark:border-yellow-800">
          Reconnecting...
        </div>
      )}

      <div className="flex-1 flex min-h-0">
        {/* Workspace Switcher */}
        <WorkspaceSwitcher />

        {/* Channel Sidebar */}
        <div
          className={cn(
            'flex-shrink-0 border-r border-gray-200 dark:border-gray-700 transition-all',
            sidebarCollapsed ? 'w-0 overflow-hidden' : 'w-64'
          )}
        >
          <ChannelSidebar workspaceId={workspaceId} />
        </div>

        {/* Main Content */}
        <div className="flex-1 flex flex-col min-w-0">
          <Outlet />
        </div>

        {/* Thread Panel */}
        {threadId && !profileUserId && (
          <ThreadPanel messageId={threadId} />
        )}

        {/* Profile Pane */}
        {profileUserId && (
          <ProfilePane userId={profileUserId} />
        )}
      </div>
    </div>
  );
}
