import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useAuth } from '../../hooks';
import { Avatar, Modal, Button, Input, toast } from '../ui';
import { useCreateWorkspace } from '../../hooks/useWorkspaces';
import { cn } from '../../lib/utils';
import { useUIStore } from '../../stores/uiStore';

export function WorkspaceSwitcher() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { workspaces, user, logout } = useAuth();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const { darkMode, toggleDarkMode } = useUIStore();

  return (
    <div className="w-16 bg-sidebar-light dark:bg-sidebar-dark flex flex-col items-center py-3 gap-3">
      {/* Workspaces */}
      <div className="flex-1 flex flex-col items-center gap-2 overflow-y-auto">
        {workspaces?.map((ws) => (
          <Link
            key={ws.id}
            to={`/workspaces/${ws.id}`}
            className={cn(
              'w-10 h-10 rounded-lg flex items-center justify-center transition-all',
              'hover:rounded-xl',
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
          className="w-10 h-10 rounded-lg flex items-center justify-center bg-transparent border-2 border-dashed border-gray-500 text-gray-400 hover:border-gray-400 hover:text-gray-300 transition-colors"
          title="Create workspace"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>

      {/* Bottom section */}
      <div className="flex flex-col items-center gap-2">
        {/* Dark mode toggle */}
        <button
          onClick={toggleDarkMode}
          className="w-10 h-10 rounded-lg flex items-center justify-center text-gray-400 hover:bg-gray-700 transition-colors"
          title={darkMode ? 'Light mode' : 'Dark mode'}
        >
          {darkMode ? (
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
          ) : (
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          )}
        </button>

        {/* User menu */}
        <button
          onClick={() => logout()}
          className="relative"
          title="Logout"
        >
          <Avatar
            src={user?.avatar_url}
            name={user?.display_name || 'User'}
            size="md"
            status="online"
          />
        </button>
      </div>

      <CreateWorkspaceModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
      />
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
          required
        />

        <Input
          label="Workspace URL"
          value={slug}
          onChange={(e) => setSlug(e.target.value)}
          placeholder="my-workspace"
          required
        />

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
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
