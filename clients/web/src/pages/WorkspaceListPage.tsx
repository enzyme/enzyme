import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth, useCreateWorkspace } from '../hooks';
import { Button, Input, Modal, toast, Spinner } from '../components/ui';

export function WorkspaceListPage() {
  const { workspaces, isLoading, logout, user } = useAuth();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-xl font-bold text-gray-900 dark:text-white">Feather</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {user?.display_name}
            </span>
            <Button variant="ghost" size="sm" onClick={() => logout()}>
              Logout
            </Button>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="max-w-4xl mx-auto px-4 py-8">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            Your Workspaces
          </h2>
          <Button onClick={() => setIsCreateModalOpen(true)}>
            Create Workspace
          </Button>
        </div>

        {workspaces && workspaces.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2">
            {workspaces.map((workspace) => (
              <Link
                key={workspace.id}
                to={`/workspaces/${workspace.id}`}
                className="block p-6 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-500 dark:hover:border-primary-500 transition-colors"
              >
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 rounded-lg bg-primary-500 flex items-center justify-center text-white font-bold text-lg">
                    {workspace.icon_url ? (
                      <img
                        src={workspace.icon_url}
                        alt={workspace.name}
                        className="w-full h-full rounded-lg object-cover"
                      />
                    ) : (
                      workspace.name.slice(0, 2).toUpperCase()
                    )}
                  </div>
                  <div>
                    <h3 className="font-semibold text-gray-900 dark:text-white">
                      {workspace.name}
                    </h3>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {workspace.role}
                    </p>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <svg
              className="w-16 h-16 mx-auto text-gray-400 mb-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
              />
            </svg>
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              No workspaces yet
            </h3>
            <p className="text-gray-500 dark:text-gray-400 mb-4">
              Create your first workspace to get started
            </p>
            <Button onClick={() => setIsCreateModalOpen(true)}>
              Create Workspace
            </Button>
          </div>
        )}
      </main>

      <CreateWorkspaceModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
      />
    </div>
  );
}

function CreateWorkspaceModal({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
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
        <p className="text-xs text-gray-500 dark:text-gray-400">
          3-50 characters, lowercase letters, numbers, and hyphens only
        </p>

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
