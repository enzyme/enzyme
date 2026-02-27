import { useState, useEffect, useCallback, useRef } from 'react';
import { Outlet, useParams } from 'react-router-dom';
import { WorkspaceSwitcher } from '../workspace/WorkspaceSwitcher';
import { ChannelSidebar } from '../channel/ChannelSidebar';
import { ThreadPanel } from '../thread/ThreadPanel';
import { ProfilePane } from '../profile/ProfilePane';
import { SearchModal } from '../search/SearchModal';
import { CommandPalette } from '../command-palette/CommandPalette';
import {
  WorkspaceSettingsModal,
  type WorkspaceSettingsTab,
} from '../settings/WorkspaceSettingsModal';
import { BanScreen } from '../moderation/BanModal';
import { useSSE, useAuth } from '../../hooks';
import { useThreadPanel, useProfilePanel } from '../../hooks/usePanel';
import { useSidebar } from '../../hooks/useSidebar';
import { useResizableWidth } from '../../hooks/useResizableWidth';
import { recordChannelVisit } from '../../lib/recentChannels';

function Divider({ ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className="w-1 flex-shrink-0 cursor-col-resize bg-transparent transition-colors hover:bg-blue-200 active:bg-blue-300 dark:hover:bg-blue-800 dark:active:bg-blue-700"
      {...props}
    />
  );
}

export function AppLayout() {
  const { workspaceId, channelId } = useParams<{ workspaceId: string; channelId: string }>();
  const { isReconnecting } = useSSE(workspaceId);
  const { workspaces } = useAuth();
  const currentWorkspace = workspaces?.find((ws) => ws.id === workspaceId);
  const { threadId } = useThreadPanel();
  const { profileUserId } = useProfilePanel();
  const { collapsed: sidebarCollapsed } = useSidebar();
  const rightPanelOpen = Boolean(threadId || profileUserId);

  const { width: sidebarWidth, dividerProps: sidebarDividerProps } = useResizableWidth(
    'enzyme:sidebar-width',
    256,
    180,
    400,
  );

  const { width: rightPanelWidth, dividerProps: rightPanelDividerProps } = useResizableWidth(
    'enzyme:right-panel-width',
    384,
    300,
    600,
    'left',
  );

  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [searchInitialQuery, setSearchInitialQuery] = useState('');
  const [isCommandPaletteOpen, setIsCommandPaletteOpen] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isNewDMModalOpen, setIsNewDMModalOpen] = useState(false);
  const [isWorkspaceSettingsOpen, setIsWorkspaceSettingsOpen] = useState(false);
  const [workspaceSettingsTab, setWorkspaceSettingsTab] = useState<WorkspaceSettingsTab>('general');
  const [settingsWorkspaceId, setSettingsWorkspaceId] = useState<string>('');

  const handleOpenSearch = useCallback((initialQuery?: string) => {
    setSearchInitialQuery(initialQuery ?? '');
    setIsSearchOpen(true);
  }, []);

  const handleOpenWorkspaceSettings = useCallback((wsId: string, tab?: WorkspaceSettingsTab) => {
    setSettingsWorkspaceId(wsId);
    setWorkspaceSettingsTab(tab ?? 'general');
    setIsWorkspaceSettingsOpen(true);
  }, []);

  const handleCreateChannel = useCallback(() => {
    setIsCreateModalOpen(true);
  }, []);

  const handleNewDM = useCallback(() => {
    setIsNewDMModalOpen(true);
  }, []);

  // Record channel visits for recent channels
  const prevChannelRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (workspaceId && channelId && channelId !== prevChannelRef.current) {
      prevChannelRef.current = channelId;
      recordChannelVisit(workspaceId, channelId);
    }
  }, [workspaceId, channelId]);

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd+K / Ctrl+K — toggle command palette
      if ((e.metaKey || e.ctrlKey) && e.key === 'k' && !e.shiftKey) {
        e.preventDefault();
        setIsCommandPaletteOpen((prev) => !prev);
      }
      // Cmd+Shift+F / Ctrl+Shift+F — open search
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.code === 'KeyF') {
        e.preventDefault();
        handleOpenSearch();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleOpenSearch]);

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

        {currentWorkspace?.ban ? (
          <BanScreen workspace={currentWorkspace} />
        ) : (
          <>
            {/* Channel Sidebar */}
            <div
              className="flex-shrink-0 overflow-hidden"
              style={{ width: sidebarCollapsed ? 0 : sidebarWidth }}
            >
              <div
                className="h-full border-r border-gray-200 dark:border-gray-700"
                style={{ width: sidebarWidth }}
              >
                <ChannelSidebar
                  workspaceId={workspaceId}
                  onSearchClick={() => setIsCommandPaletteOpen(true)}
                  onOpenWorkspaceSettings={handleOpenWorkspaceSettings}
                  onCreateChannel={handleCreateChannel}
                  onNewDM={handleNewDM}
                  isCreateModalOpen={isCreateModalOpen}
                  onCloseCreateModal={() => setIsCreateModalOpen(false)}
                  isNewDMModalOpen={isNewDMModalOpen}
                  onCloseNewDMModal={() => setIsNewDMModalOpen(false)}
                />
              </div>
            </div>

            {!sidebarCollapsed && <Divider {...sidebarDividerProps} />}

            {/* Main Content */}
            <div className="flex min-w-0 flex-1 flex-col">
              <Outlet />
            </div>

            {/* Right Panel (Thread / Profile) */}
            {rightPanelOpen && (
              <>
                <Divider {...rightPanelDividerProps} />
                <div className="flex-shrink-0" style={{ width: rightPanelWidth }}>
                  {threadId && !profileUserId && <ThreadPanel messageId={threadId} />}
                  {profileUserId && <ProfilePane userId={profileUserId} />}
                </div>
              </>
            )}
          </>
        )}
      </div>

      {!currentWorkspace?.ban && (
        <>
          {/* Command Palette */}
          <CommandPalette
            isOpen={isCommandPaletteOpen}
            onClose={() => setIsCommandPaletteOpen(false)}
            onOpenSearch={handleOpenSearch}
            onCreateChannel={handleCreateChannel}
            onNewDM={handleNewDM}
            onOpenWorkspaceSettings={handleOpenWorkspaceSettings}
          />

          {/* Search Modal */}
          <SearchModal
            isOpen={isSearchOpen}
            onClose={() => setIsSearchOpen(false)}
            initialChannelId={channelId}
            initialQuery={searchInitialQuery}
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
        </>
      )}
    </div>
  );
}
