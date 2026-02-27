import { useState, useEffect, useLayoutEffect, useCallback, useRef } from 'react';
import { Outlet, useParams } from 'react-router-dom';
import { Group, Panel, Separator, useDefaultLayout, usePanelRef } from 'react-resizable-panels';
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
import { recordChannelVisit } from '../../lib/recentChannels';

export function AppLayout() {
  const { workspaceId, channelId } = useParams<{ workspaceId: string; channelId: string }>();
  const { isReconnecting } = useSSE(workspaceId);
  const { workspaces } = useAuth();
  const currentWorkspace = workspaces?.find((ws) => ws.id === workspaceId);
  const { threadId } = useThreadPanel();
  const { profileUserId } = useProfilePanel();
  const { collapsed: sidebarCollapsed, setCollapsed: setSidebarCollapsed } = useSidebar();
  const sidebarPanelRef = usePanelRef();
  const rightPanelRef = usePanelRef();
  const rightPanelOpen = Boolean(threadId || profileUserId);

  // Persist panel layout in localStorage
  const { defaultLayout, onLayoutChanged } = useDefaultLayout({
    id: 'enzyme:layout',
    storage: localStorage,
  });

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

  // Sync sidebar panel collapse state with useSidebar hook via onResize
  const handleSidebarResize = useCallback(
    (size: { asPercentage: number }) => {
      const isNowCollapsed = size.asPercentage === 0;
      if (isNowCollapsed && !sidebarCollapsed) {
        setSidebarCollapsed(true);
      } else if (!isNowCollapsed && sidebarCollapsed) {
        setSidebarCollapsed(false);
      }
    },
    [sidebarCollapsed, setSidebarCollapsed],
  );

  // Sync sidebar toggle (from keyboard shortcut) to panel ref
  useLayoutEffect(() => {
    const panel = sidebarPanelRef.current;
    if (!panel) return;
    if (sidebarCollapsed && !panel.isCollapsed()) {
      panel.collapse();
    } else if (!sidebarCollapsed && panel.isCollapsed()) {
      panel.expand();
    }
  }, [sidebarCollapsed, sidebarPanelRef]);

  // Sync URL params to right panel collapse/expand (after initial mount)
  useLayoutEffect(() => {
    const panel = rightPanelRef.current;
    if (!panel) return;
    if (rightPanelOpen) {
      if (panel.isCollapsed()) panel.expand();
    } else {
      if (!panel.isCollapsed()) panel.collapse();
    }
  }, [rightPanelOpen, rightPanelRef]);

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
          <Group
            orientation="horizontal"
            defaultLayout={defaultLayout}
            onLayoutChanged={onLayoutChanged}
            id="enzyme:layout"
          >
            {/* Channel Sidebar */}
            <Panel
              id="sidebar"
              panelRef={sidebarPanelRef}
              collapsible
              collapsedSize={0}
              minSize={15}
              maxSize={30}
              defaultSize={sidebarCollapsed ? 0 : 20}
              onResize={handleSidebarResize}
            >
              <div className="h-full overflow-hidden border-r border-gray-200 dark:border-gray-700">
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
            </Panel>

            <Separator className="w-1 cursor-col-resize bg-transparent transition-colors data-[separator=hover]:bg-blue-500/30 data-[separator=active]:bg-blue-500/50" />

            {/* Main Content */}
            <Panel id="content" minSize={30}>
              <div className="flex h-full min-w-0 flex-col">
                <Outlet />
              </div>
            </Panel>

            <Separator className="w-1 cursor-col-resize bg-transparent transition-colors data-[separator=hover]:bg-blue-500/30 data-[separator=active]:bg-blue-500/50" />

            {/* Right Panel (Thread / Profile) */}
            <Panel
              id="right-panel"
              panelRef={rightPanelRef}
              collapsible
              collapsedSize={0}
              minSize={20}
              maxSize={45}
              defaultSize={rightPanelOpen ? 30 : 0}
            >
              {threadId && !profileUserId && (
                <ThreadPanel messageId={threadId} />
              )}
              {profileUserId && (
                <ProfilePane userId={profileUserId} />
              )}
            </Panel>
          </Group>
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
