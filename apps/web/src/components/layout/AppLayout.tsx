import { useState, useEffect, useCallback } from 'react';
import { Outlet, useParams } from 'react-router-dom';
import { WorkspaceSwitcher } from '../workspace/WorkspaceSwitcher';
import { ChannelSidebar } from '../channel/ChannelSidebar';
import { ThreadPanel } from '../thread/ThreadPanel';
import { ProfilePane } from '../profile/ProfilePane';
import { SearchModal } from '../search/SearchModal';
import {
  WorkspaceSettingsModal,
  type WorkspaceSettingsTab,
} from '../settings/WorkspaceSettingsModal';
import { useSSE } from '../../hooks';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { useSidebar } from '../../hooks/useSidebar';
import { cn } from '../../lib/utils';

export function AppLayout() {
  const { workspaceId, channelId } = useParams<{ workspaceId: string; channelId: string }>();
  const { isReconnecting } = useSSE(workspaceId);
  const { threadId } = useThreadPanel();
  const { profileUserId } = useProfilePanel();
  const { collapsed: sidebarCollapsed } = useSidebar();
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [isWorkspaceSettingsOpen, setIsWorkspaceSettingsOpen] = useState(false);
  const [workspaceSettingsTab, setWorkspaceSettingsTab] = useState<WorkspaceSettingsTab>('general');
  const [settingsWorkspaceId, setSettingsWorkspaceId] = useState<string>('');
  const handleOpenSearch = useCallback(() => {
    setIsSearchOpen(true);
  }, []);

  const handleOpenWorkspaceSettings = useCallback((wsId: string, tab?: WorkspaceSettingsTab) => {
    setSettingsWorkspaceId(wsId);
    setWorkspaceSettingsTab(tab ?? 'general');
    setIsWorkspaceSettingsOpen(true);
  }, []);

  // Global Cmd+K / Ctrl+K keyboard shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setIsSearchOpen(true);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  return (
    <div className="flex h-screen flex-col bg-white dark:bg-gray-900">
      {/* Connection Status - full width */}
      {isReconnecting && workspaceId && (
        <div className="flex-shrink-0 border-b border-yellow-200 bg-yellow-100 px-4 py-1.5 text-center text-sm text-yellow-800 dark:border-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-200">
          Reconnecting...
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {/* Workspace Switcher */}
        <WorkspaceSwitcher onOpenWorkspaceSettings={handleOpenWorkspaceSettings} />

        {/* Channel Sidebar */}
        <div
          className={cn(
            'flex-shrink-0 border-r border-gray-200 transition-all dark:border-gray-700',
            sidebarCollapsed ? 'w-0 overflow-hidden' : 'w-64',
          )}
        >
          <ChannelSidebar
            workspaceId={workspaceId}
            onSearchClick={handleOpenSearch}
            onOpenWorkspaceSettings={handleOpenWorkspaceSettings}
          />
        </div>

        {/* Main Content */}
        <div className="flex min-w-0 flex-1 flex-col">
          <Outlet />
        </div>

        {/* Thread Panel */}
        {threadId && !profileUserId && <ThreadPanel messageId={threadId} />}

        {/* Profile Pane */}
        {profileUserId && <ProfilePane userId={profileUserId} />}
      </div>

      {/* Search Modal */}
      <SearchModal
        isOpen={isSearchOpen}
        onClose={() => setIsSearchOpen(false)}
        initialChannelId={channelId}
      />

      {/* Workspace Settings Modal */}
      {settingsWorkspaceId && (
        <WorkspaceSettingsModal
          isOpen={isWorkspaceSettingsOpen}
          onClose={() => setIsWorkspaceSettingsOpen(false)}
          workspaceId={settingsWorkspaceId}
          defaultTab={workspaceSettingsTab}
        />
      )}
    </div>
  );
}
