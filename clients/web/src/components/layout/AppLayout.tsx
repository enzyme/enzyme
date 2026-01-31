import { Outlet, useParams } from 'react-router-dom';
import { WorkspaceSwitcher } from '../workspace/WorkspaceSwitcher';
import { ChannelSidebar } from '../channel/ChannelSidebar';
import { ThreadPanel } from '../thread/ThreadPanel';
import { useSSE } from '../../hooks';
import { useUIStore } from '../../stores/uiStore';
import { cn } from '../../lib/utils';

export function AppLayout() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { isConnected } = useSSE(workspaceId);
  const { activeThreadId, sidebarCollapsed } = useUIStore();

  return (
    <div className="h-screen flex bg-white dark:bg-gray-900">
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
        {/* Connection Status */}
        {!isConnected && workspaceId && (
          <div className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 text-sm px-4 py-2 text-center">
            Reconnecting...
          </div>
        )}

        <Outlet />
      </div>

      {/* Thread Panel */}
      {activeThreadId && (
        <ThreadPanel messageId={activeThreadId} />
      )}
    </div>
  );
}
