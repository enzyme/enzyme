import { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { LoginForm } from '../components/auth';
import { useAuth, useCreateWorkspace } from '../hooks';
import { Spinner, Button, Input, toast } from '../components/ui';

export function LoginPage() {
  const { isAuthenticated, isLoading, workspaces } = useAuth();
  const [workspaceName, setWorkspaceName] = useState('');
  const createWorkspace = useCreateWorkspace();

  if (isAuthenticated) {
    // Check for pending invite
    const pendingInvite = sessionStorage.getItem('pendingInvite');
    if (pendingInvite) {
      // Don't remove here - StrictMode double-renders would clear it before redirect
      // AcceptInvitePage will clear it after processing
      return <Navigate to={`/invites/${pendingInvite}`} replace />;
    }
    // Redirect to first workspace
    if (workspaces && workspaces.length > 0) {
      return <Navigate to={`/workspaces/${workspaces[0].id}`} replace />;
    }
    // Edge case: authenticated but no workspaces - show create workspace form
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="w-full max-w-md">
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
              Create a Workspace
            </h1>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              You need a workspace to get started
            </p>
          </div>

          <form
            onSubmit={async (e) => {
              e.preventDefault();
              if (!workspaceName.trim()) return;
              try {
                await createWorkspace.mutateAsync({ name: workspaceName.trim() });
              } catch (err) {
                toast(err instanceof Error ? err.message : 'Failed to create workspace', 'error');
              }
            }}
            className="space-y-4"
          >
            <Input
              label="Workspace name"
              value={workspaceName}
              onChange={(e) => setWorkspaceName(e.target.value)}
              placeholder="My Workspace"
              isRequired
            />

            <Button
              type="submit"
              className="w-full"
              isLoading={createWorkspace.isPending}
            >
              Create Workspace
            </Button>
          </form>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
      <LoginForm />
    </div>
  );
}
