import { useState, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button as AriaButton } from 'react-aria-components';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragOverlay,
} from '@dnd-kit/core';
import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import {
  PlusIcon,
  SunIcon,
  MoonIcon,
  UserIcon,
  ArrowRightStartOnRectangleIcon,
  ComputerDesktopIcon,
  CheckIcon,
} from '@heroicons/react/24/outline';
import { useAuth, useChannels } from '../../hooks';
import {
  Avatar,
  Modal,
  Button,
  Input,
  toast,
  Tooltip,
  Menu,
  MenuItem,
  SubmenuTrigger,
  MenuSection,
  MenuHeader,
  MenuSeparator,
} from '../ui';
import {
  useCreateWorkspace,
  useReorderWorkspaces,
  useWorkspaceNotifications,
} from '../../hooks/useWorkspaces';
import { useDarkMode } from '../../hooks/useDarkMode';
import { useProfilePanel } from '../../hooks/usePanel';
import { cn, getAvatarColor } from '../../lib/utils';
import type { WorkspaceSummary, WorkspaceNotificationSummary } from '@enzyme/api-client';
import { WorkspaceContextMenu } from './WorkspaceContextMenu';

import type { WorkspaceSettingsTab } from '../settings/WorkspaceSettingsModal';

interface WorkspaceSwitcherProps {
  onOpenWorkspaceSettings: (workspaceId: string, tab?: WorkspaceSettingsTab) => void;
}

export function WorkspaceSwitcher({ onOpenWorkspaceSettings }: WorkspaceSwitcherProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const navigate = useNavigate();
  const { workspaces: serverWorkspaces } = useAuth();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [activeWorkspace, setActiveWorkspace] = useState<WorkspaceSummary | null>(null);
  const reorderWorkspaces = useReorderWorkspaces();
  const { notificationMap } = useWorkspaceNotifications();

  // Local state for immediate visual updates during drag
  const [localWorkspaces, setLocalWorkspaces] = useState<WorkspaceSummary[]>([]);
  const [prevServerWorkspaces, setPrevServerWorkspaces] = useState<
    WorkspaceSummary[] | undefined
  >();

  // Sync local state with server state when server data changes (React-recommended pattern)
  if (serverWorkspaces !== prevServerWorkspaces) {
    setPrevServerWorkspaces(serverWorkspaces);
    if (serverWorkspaces) {
      setLocalWorkspaces(serverWorkspaces);
    }
  }

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const handleDragStart = (event: DragStartEvent) => {
    const workspace = localWorkspaces.find((ws) => ws.id === event.active.id);
    if (workspace) {
      setActiveWorkspace(workspace);
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveWorkspace(null);

    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = localWorkspaces.findIndex((ws) => ws.id === active.id);
    const newIndex = localWorkspaces.findIndex((ws) => ws.id === over.id);

    if (oldIndex !== -1 && newIndex !== -1) {
      // Update local state immediately for smooth visual transition
      const newOrder = arrayMove(localWorkspaces, oldIndex, newIndex);
      setLocalWorkspaces(newOrder);

      // Persist to server
      reorderWorkspaces.mutate(newOrder.map((ws) => ws.id));
    }
  };

  const workspaceIds = localWorkspaces.map((ws) => ws.id);

  return (
    <div className="flex w-full flex-col border-r border-gray-200 bg-white pb-4 md:w-16 md:items-center md:gap-4 md:pt-3 dark:border-gray-700 dark:bg-gray-900">
      <h2 className="border-b border-gray-200 px-3 py-3 font-semibold text-gray-900 md:hidden dark:border-gray-700 dark:text-white">
        Workspaces
      </h2>
      {/* Workspaces */}
      <div className="flex flex-1 flex-col gap-1 overflow-y-auto px-3 py-2 md:items-center md:gap-3 md:px-2">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={workspaceIds} strategy={verticalListSortingStrategy}>
            {localWorkspaces.map((ws) => (
              <SortableWorkspaceItem
                key={ws.id}
                workspace={ws}
                isActive={ws.id === workspaceId}
                onPress={() => navigate(`/workspaces/${ws.id}`)}
                notificationMap={notificationMap}
                onOpenWorkspaceSettings={onOpenWorkspaceSettings}
                canInvite={ws.role === 'owner' || ws.role === 'admin'}
              />
            ))}
          </SortableContext>
          <DragOverlay>
            {activeWorkspace && (
              <WorkspaceItemContent
                workspace={activeWorkspace}
                isActive={activeWorkspace.id === workspaceId}
                isDragOverlay
                notificationMap={notificationMap}
              />
            )}
          </DragOverlay>
        </DndContext>

        {/* Add Workspace Button */}
        <Tooltip content="Add workspace" placement="right">
          <AriaButton
            onPress={() => setIsCreateModalOpen(true)}
            className="flex h-10 cursor-pointer items-center gap-3 rounded-lg px-2 py-1 text-gray-500 transition-colors hover:bg-gray-100 md:h-auto md:justify-center md:p-0 dark:text-gray-400 dark:hover:bg-gray-800"
          >
            <span className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-gray-200 dark:bg-gray-700">
              <PlusIcon className="h-4 w-4" />
            </span>
            <span className="text-sm font-medium md:hidden">Add Workspace</span>
          </AriaButton>
        </Tooltip>
      </div>

      {/* Bottom section */}
      <div className="border-t border-gray-200 px-3 pt-3 md:border-0 md:px-0 md:pt-0 dark:border-gray-700">
        <UserMenu />
      </div>

      <CreateWorkspaceModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
      />
    </div>
  );
}

interface SortableWorkspaceItemProps {
  workspace: WorkspaceSummary;
  isActive: boolean;
  onPress: () => void;
  notificationMap: Map<string, WorkspaceNotificationSummary>;
  onOpenWorkspaceSettings: (workspaceId: string, tab?: WorkspaceSettingsTab) => void;
  canInvite: boolean;
}

function SortableWorkspaceItem({
  workspace,
  isActive,
  onPress,
  notificationMap,
  onOpenWorkspaceSettings,
  canInvite,
}: SortableWorkspaceItemProps) {
  const { listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: workspace.id,
  });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    cursor: isDragging ? 'grabbing' : 'grab',
  };

  // Only forward pointer listeners from dnd-kit — keyboard Enter/Space
  // should navigate (via AriaButton onPress), not start a drag.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { onKeyDown: _dndKeyDown, ...pointerListeners } = (listeners || {}) as Record<
    string,
    unknown
  >;

  return (
    <WorkspaceContextMenu
      workspaceId={workspace.id}
      onOpenWorkspaceSettings={() => onOpenWorkspaceSettings(workspace.id)}
      onOpenInvite={() => onOpenWorkspaceSettings(workspace.id, 'invite')}
      canInvite={canInvite}
    >
      {(onContextMenu, isMenuOpen) => (
        <Tooltip content={workspace.name} placement="right">
          <AriaButton
            ref={setNodeRef}
            style={style}
            onPress={onPress}
            aria-label={workspace.name}
            className={cn(
              'flex h-10 cursor-grab items-center gap-3 rounded-lg px-2 py-1 transition-colors outline-none hover:bg-gray-100 md:h-auto md:p-0 dark:hover:bg-gray-800',
              isDragging && 'cursor-grabbing opacity-50',
              isActive && 'bg-gray-100 dark:bg-gray-800',
            )}
            {...(pointerListeners as React.DOMAttributes<Element>)}
            onContextMenu={onContextMenu}
          >
            <WorkspaceItemContent
              workspace={workspace}
              isActive={isActive}
              isMenuOpen={isMenuOpen}
              notificationMap={notificationMap}
            />
            <span className="truncate text-sm font-medium text-gray-900 md:hidden dark:text-white">
              {workspace.name}
            </span>
          </AriaButton>
        </Tooltip>
      )}
    </WorkspaceContextMenu>
  );
}

interface WorkspaceItemContentProps {
  workspace: WorkspaceSummary;
  isActive: boolean;
  isDragOverlay?: boolean;
  isMenuOpen?: boolean;
  notificationMap: Map<string, WorkspaceNotificationSummary>;
}

function WorkspaceItemContent({
  workspace,
  isActive,
  isDragOverlay,
  isMenuOpen,
  notificationMap,
}: WorkspaceItemContentProps) {
  // Fetch channel data for the active workspace (real-time via SSE)
  const { data: channelsData } = useChannels(isActive ? workspace.id : undefined);

  const { hasUnread, totalNotifications } = useMemo(() => {
    if (isActive) {
      // Active workspace: use channel data for real-time accuracy
      if (!channelsData?.channels) return { hasUnread: false, totalNotifications: 0 };
      return {
        hasUnread: channelsData.channels.some((c) => c.unread_count > 0),
        totalNotifications: channelsData.channels.reduce((sum, c) => sum + c.notification_count, 0),
      };
    }
    // Non-active workspace: use polled notification summaries
    const summary = notificationMap.get(workspace.id);
    if (!summary) return { hasUnread: false, totalNotifications: 0 };
    return {
      hasUnread: summary.unread_count > 0,
      totalNotifications: summary.notification_count,
    };
  }, [isActive, channelsData, notificationMap, workspace.id]);

  return (
    <div className="relative">
      <div
        className={cn(
          'flex h-8 w-8 items-center justify-center rounded-lg transition-colors',
          isActive ? 'ring-2 ring-gray-900 dark:ring-white' : '',
          workspace.icon_url ? '' : `${getAvatarColor(workspace.id)} hover:opacity-80`,
          isDragOverlay && 'cursor-grabbing shadow-lg',
          isMenuOpen && !isActive && 'ring-2 ring-gray-400 dark:ring-gray-500',
        )}
      >
        {workspace.icon_url ? (
          <img
            src={workspace.icon_url}
            alt={workspace.name}
            className="h-full w-full rounded-lg object-cover"
          />
        ) : (
          <span className="text-xs font-semibold text-white">
            {workspace.name.slice(0, 2).toUpperCase()}
          </span>
        )}
      </div>
      {!isActive && totalNotifications > 0 && (
        <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white ring-2 ring-[var(--avatar-ring)]">
          {totalNotifications > 99 ? '99+' : totalNotifications}
        </span>
      )}
      {!isActive && totalNotifications === 0 && hasUnread && (
        <span className="absolute -top-0.5 -right-0.5 h-1.5 w-1.5 rounded-full bg-gray-900 ring-2 ring-[var(--avatar-ring)] dark:bg-gray-100" />
      )}
    </div>
  );
}

function UserMenu() {
  const { user, logout } = useAuth();
  const { openProfile } = useProfilePanel();
  const navigate = useNavigate();
  const { mode, setMode } = useDarkMode();

  const handleLogout = async () => {
    try {
      await logout();
      navigate('/login');
    } catch {
      // Ignore logout errors - still redirect
      navigate('/login');
    }
  };

  // Get icon for current mode
  const ThemeIcon = mode === 'system' ? ComputerDesktopIcon : mode === 'light' ? SunIcon : MoonIcon;

  const themeModeLabel = mode === 'system' ? 'System' : mode === 'light' ? 'Light' : 'Dark';

  return (
    <Menu
      align="start"
      placement="top"
      trigger={
        <AriaButton className="flex w-full cursor-pointer items-center gap-3 rounded-lg px-2 py-1 transition-colors outline-none hover:bg-gray-100 hover:opacity-80 md:w-auto md:p-0 dark:hover:bg-gray-800">
          <Avatar
            src={user?.avatar_url}
            gravatarSrc={user?.gravatar_url}
            name={user?.display_name || 'User'}
            id={user?.id}
            size="md"
            status="online"
          />
          <span className="truncate text-sm font-medium text-gray-900 md:hidden dark:text-white">
            {user?.display_name}
          </span>
        </AriaButton>
      }
    >
      <MenuSection>
        <MenuHeader>
          <p className="truncate font-medium text-gray-900 dark:text-white">{user?.display_name}</p>
          <p className="truncate text-sm text-gray-500 dark:text-gray-400">{user?.email}</p>
        </MenuHeader>

        <MenuItem
          onAction={() => user?.id && openProfile(user.id)}
          icon={<UserIcon className="h-4 w-4" />}
        >
          View Profile
        </MenuItem>

        {/* Theme selector with submenu */}
        <SubmenuTrigger label={`Theme: ${themeModeLabel}`} icon={<ThemeIcon className="h-4 w-4" />}>
          <MenuItem
            onAction={() => setMode('system')}
            icon={<ComputerDesktopIcon className="h-4 w-4" />}
          >
            <span className="flex-1">System</span>
            {mode === 'system' && <CheckIcon className="h-4 w-4" />}
          </MenuItem>
          <MenuItem onAction={() => setMode('light')} icon={<SunIcon className="h-4 w-4" />}>
            <span className="flex-1">Light</span>
            {mode === 'light' && <CheckIcon className="h-4 w-4" />}
          </MenuItem>
          <MenuItem onAction={() => setMode('dark')} icon={<MoonIcon className="h-4 w-4" />}>
            <span className="flex-1">Dark</span>
            {mode === 'dark' && <CheckIcon className="h-4 w-4" />}
          </MenuItem>
        </SubmenuTrigger>

        <MenuSeparator />

        <MenuItem
          onAction={handleLogout}
          variant="danger"
          icon={<ArrowRightStartOnRectangleIcon className="h-4 w-4" />}
        >
          Log Out
        </MenuItem>
      </MenuSection>
    </Menu>
  );
}

function CreateWorkspaceModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [name, setName] = useState('');
  const createWorkspace = useCreateWorkspace();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      await createWorkspace.mutateAsync({ name });
      toast('Workspace created!', 'success');
      onClose();
      setName('');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to create workspace', 'error');
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Create Workspace">
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Workspace Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My Workspace"
          isRequired
        />

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onPress={onClose}>
            Cancel
          </Button>
          <Button type="submit" isLoading={createWorkspace.isPending}>
            Create
          </Button>
        </div>
      </form>
    </Modal>
  );
}
