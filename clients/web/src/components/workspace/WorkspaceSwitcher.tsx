import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Button as AriaButton } from "react-aria-components";
import {
  PlusIcon,
  SunIcon,
  MoonIcon,
  UserIcon,
  Cog6ToothIcon,
  UserPlusIcon,
  ServerStackIcon,
  ArrowRightStartOnRectangleIcon,
  ComputerDesktopIcon,
  CheckIcon,
} from "@heroicons/react/24/outline";
import { useAuth } from "../../hooks";
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
} from "../ui";
import { useCreateWorkspace } from "../../hooks/useWorkspaces";
import { useDarkMode } from "../../hooks/useDarkMode";
import { useProfilePanel } from "../../hooks/usePanel";
import { cn, getAvatarColor } from "../../lib/utils";

export function WorkspaceSwitcher() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const navigate = useNavigate();
  const { workspaces } = useAuth();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  return (
    <div className="w-16 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 flex flex-col items-center py-4 gap-4">
      {/* Workspaces */}
      <div className="flex-1 flex flex-col items-center gap-3 overflow-y-auto p-1">
        {workspaces?.map((ws) => (
          <Tooltip key={ws.id} content={ws.name} placement="right">
            <AriaButton
              onPress={() => navigate(`/workspaces/${ws.id}`)}
              className={cn(
                "w-8 h-8 rounded-lg flex items-center justify-center transition-colors",
                ws.id === workspaceId
                  ? "ring-2 ring-gray-900 dark:ring-white"
                  : "",
                ws.icon_url ? "" : `${getAvatarColor(ws.id)} hover:opacity-80`,
              )}
            >
              {ws.icon_url ? (
                <img
                  src={ws.icon_url}
                  alt={ws.name}
                  className="w-full h-full rounded-lg object-cover"
                />
              ) : (
                <span className="text-white font-semibold text-xs">
                  {ws.name.slice(0, 2).toUpperCase()}
                </span>
              )}
            </AriaButton>
          </Tooltip>
        ))}

        {/* Add Workspace Button */}
        <Tooltip content="Add workspace" placement="right">
          <AriaButton
            onPress={() => setIsCreateModalOpen(true)}
            className="w-8 h-8 rounded-lg flex items-center justify-center bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 hover:bg-gray-300 dark:hover:bg-gray-600 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          >
            <PlusIcon className="w-4 h-4" />
          </AriaButton>
        </Tooltip>
      </div>

      {/* Bottom section */}
      <div className="flex flex-col items-center gap-3">
        {/* User menu */}
        <UserMenu />
      </div>

      <CreateWorkspaceModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
      />
    </div>
  );
}

function UserMenu() {
  const { user, logout } = useAuth();
  const { openProfile } = useProfilePanel();
  const { workspaceId } = useParams<{ workspaceId: string }>();
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
  const ThemeIcon =
    mode === "system"
      ? ComputerDesktopIcon
      : mode === "light"
        ? SunIcon
        : MoonIcon;

  const themeModeLabel =
    mode === "system" ? "System" : mode === "light" ? "Light" : "Dark";

  return (
    <Menu
      align="start"
      placement="top"
      trigger={
        <AriaButton className="outline-none">
          <Avatar
            src={user?.avatar_url}
            name={user?.display_name || "User"}
            id={user?.id}
            size="md"
            status="online"
          />
        </AriaButton>
      }
    >
      <MenuSection>
        <MenuHeader>
          <p className="font-medium text-gray-900 dark:text-white truncate">
            {user?.display_name}
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
            {user?.email}
          </p>
        </MenuHeader>

        <MenuItem
          onAction={() => user?.id && openProfile(user.id)}
          icon={<UserIcon className="w-4 h-4" />}
        >
          View Profile
        </MenuItem>

        {/* Theme selector with submenu */}
        <SubmenuTrigger
          label={`Theme: ${themeModeLabel}`}
          icon={<ThemeIcon className="w-4 h-4" />}
        >
          <MenuItem
            onAction={() => setMode("system")}
            icon={<ComputerDesktopIcon className="w-4 h-4" />}
          >
            <span className="flex-1">System</span>
            {mode === "system" && <CheckIcon className="w-4 h-4" />}
          </MenuItem>
          <MenuItem
            onAction={() => setMode("light")}
            icon={<SunIcon className="w-4 h-4" />}
          >
            <span className="flex-1">Light</span>
            {mode === "light" && <CheckIcon className="w-4 h-4" />}
          </MenuItem>
          <MenuItem
            onAction={() => setMode("dark")}
            icon={<MoonIcon className="w-4 h-4" />}
          >
            <span className="flex-1">Dark</span>
            {mode === "dark" && <CheckIcon className="w-4 h-4" />}
          </MenuItem>
        </SubmenuTrigger>

        {workspaceId && (
          <>
            <MenuItem
              onAction={() => navigate(`/workspaces/${workspaceId}/settings`)}
              icon={<Cog6ToothIcon className="w-4 h-4" />}
            >
              Workspace Settings
            </MenuItem>
            <MenuItem
              onAction={() => navigate(`/workspaces/${workspaceId}/invite`)}
              icon={<UserPlusIcon className="w-4 h-4" />}
            >
              Invite People
            </MenuItem>
          </>
        )}

        <MenuItem
          onAction={() => navigate("/settings")}
          icon={<ServerStackIcon className="w-4 h-4" />}
        >
          Server Settings
        </MenuItem>

        <MenuSeparator />

        <MenuItem
          onAction={handleLogout}
          variant="danger"
          icon={<ArrowRightStartOnRectangleIcon className="w-4 h-4" />}
        >
          Log Out
        </MenuItem>
      </MenuSection>
    </Menu>
  );
}

function CreateWorkspaceModal({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const [name, setName] = useState("");
  const createWorkspace = useCreateWorkspace();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      await createWorkspace.mutateAsync({ name });
      toast("Workspace created!", "success");
      onClose();
      setName("");
    } catch (err) {
      toast(
        err instanceof Error ? err.message : "Failed to create workspace",
        "error",
      );
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
