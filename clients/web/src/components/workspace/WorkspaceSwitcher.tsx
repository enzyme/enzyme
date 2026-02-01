import { useState, useRef, useEffect } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  PlusIcon,
  SunIcon,
  MoonIcon,
  UserIcon,
  Cog6ToothIcon,
  UserPlusIcon,
  ServerStackIcon,
  ArrowRightStartOnRectangleIcon,
} from '@heroicons/react/24/outline';
import { useAuth } from '../../hooks';
import { Avatar, Modal, Button, Input, toast } from '../ui';
import { useCreateWorkspace } from '../../hooks/useWorkspaces';
import { useUIStore } from '../../stores/uiStore';
import { cn } from '../../lib/utils';

export function WorkspaceSwitcher() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { workspaces } = useAuth();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const { darkMode, toggleDarkMode } = useUIStore();

  return (
    <div className="w-16 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 flex flex-col items-center py-3 gap-3">
      {/* Workspaces */}
      <div className="flex-1 flex flex-col items-center gap-2 overflow-y-auto">
        {workspaces?.map((ws) => (
          <Link
            key={ws.id}
            to={`/workspaces/${ws.id}`}
            className={cn(
              'w-10 h-10 rounded-lg flex items-center justify-center transition-all hover:rounded-xl',
              ws.id === workspaceId
                ? 'bg-primary-600 rounded-xl'
                : 'bg-gray-600 hover:bg-gray-500'
            )}
            title={ws.name}
          >
            {ws.icon_url ? (
              <img
                src={ws.icon_url}
                alt={ws.name}
                className="w-full h-full rounded-lg object-cover"
              />
            ) : (
              <span className="text-white font-semibold text-sm">
                {ws.name.slice(0, 2).toUpperCase()}
              </span>
            )}
          </Link>
        ))}

        {/* Add Workspace Button */}
        <button
          onClick={() => setIsCreateModalOpen(true)}
          className="w-10 h-10 rounded-lg flex items-center justify-center bg-transparent border-2 border-dashed border-gray-300 dark:border-gray-500 text-gray-400 hover:border-gray-400 dark:hover:border-gray-400 hover:text-gray-500 dark:hover:text-gray-300 transition-colors"
          title="Create workspace"
        >
          <PlusIcon className="w-5 h-5" />
        </button>
      </div>

      {/* Bottom section */}
      <div className="flex flex-col items-center gap-2">
        {/* Dark mode toggle */}
        <button
          onClick={toggleDarkMode}
          className="w-10 h-10 rounded-lg flex items-center justify-center text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          title={darkMode ? 'Light mode' : 'Dark mode'}
        >
          {darkMode ? (
            <SunIcon className="w-5 h-5" />
          ) : (
            <MoonIcon className="w-5 h-5" />
          )}
        </button>

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
  const { openProfile } = useUIStore();
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  // Close on click outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleLogout = async () => {
    try {
      await logout();
    } catch {
      // Ignore logout errors
    }
  };

  return (
    <div ref={menuRef} className="relative">
      <button onClick={() => setIsOpen(!isOpen)}>
        <Avatar
          src={user?.avatar_url}
          name={user?.display_name || 'User'}
          size="md"
          status="online"
        />
      </button>

      {isOpen && (
        <div className="absolute bottom-full left-0 mb-2 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50">
          {/* User info header */}
          <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700">
            <p className="font-medium text-gray-900 dark:text-white truncate">
              {user?.display_name}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
              {user?.email}
            </p>
          </div>

          {/* Menu items */}
          <div className="py-1">
            <button
              onClick={() => {
                if (user?.id) openProfile(user.id);
                setIsOpen(false);
              }}
              className="w-full px-3 py-1.5 text-left text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <UserIcon className="w-4 h-4" />
              View Profile
            </button>

            {workspaceId && (
              <>
                <Link
                  to={`/workspaces/${workspaceId}/settings`}
                  onClick={() => setIsOpen(false)}
                  className="w-full px-3 py-1.5 text-left text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                >
                  <Cog6ToothIcon className="w-4 h-4" />
                  Workspace Settings
                </Link>

                <Link
                  to={`/workspaces/${workspaceId}/invite`}
                  onClick={() => setIsOpen(false)}
                  className="w-full px-3 py-1.5 text-left text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                >
                  <UserPlusIcon className="w-4 h-4" />
                  Invite People
                </Link>
              </>
            )}

            <Link
              to="/settings"
              onClick={() => setIsOpen(false)}
              className="w-full px-3 py-1.5 text-left text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <ServerStackIcon className="w-4 h-4" />
              Server Settings
            </Link>
          </div>

          {/* Logout */}
          <div className="border-t border-gray-200 dark:border-gray-700 py-1">
            <button
              onClick={handleLogout}
              className="w-full px-3 py-1.5 text-left text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <ArrowRightStartOnRectangleIcon className="w-4 h-4" />
              Log Out
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function CreateWorkspaceModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const createWorkspace = useCreateWorkspace();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      await createWorkspace.mutateAsync({ name, slug });
      toast('Workspace created!', 'success');
      onClose();
      setName('');
      setSlug('');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to create workspace', 'error');
    }
  };

  const handleNameChange = (value: string) => {
    setName(value);
    // Auto-generate slug from name
    setSlug(
      value
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-|-$/g, '')
    );
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Create Workspace">
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Workspace Name"
          value={name}
          onChange={(e) => handleNameChange(e.target.value)}
          placeholder="My Workspace"
          isRequired
        />

        <Input
          label="Workspace URL"
          value={slug}
          onChange={(e) => setSlug(e.target.value)}
          placeholder="my-workspace"
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
