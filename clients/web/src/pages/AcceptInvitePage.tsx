import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useAcceptInvite, useAuth } from '../hooks';
import { Button, Spinner } from '../components/ui';

export function AcceptInvitePage() {
  const { code } = useParams<{ code: string }>();
  const navigate = useNavigate();
  const { isAuthenticated, isLoading: authLoading, workspaces } = useAuth();
  const acceptInvite = useAcceptInvite();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      // Save invite code and redirect to login
      sessionStorage.setItem('pendingInvite', code || '');
    }
  }, [authLoading, isAuthenticated, code]);

  const handleAccept = async () => {
    if (!code) return;

    try {
      const result = await acceptInvite.mutateAsync(code);
      // Clear pending invite now that it's been accepted
      sessionStorage.removeItem('pendingInvite');
      navigate(`/workspaces/${result.workspace.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to accept invite');
    }
  };

  // Get fallback workspace link for error state
  const fallbackWorkspaceLink = workspaces && workspaces.length > 0
    ? `/workspaces/${workspaces[0].id}`
    : '/login';

  if (authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-md w-full text-center">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
            You've been invited!
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mb-6">
            Sign in or create an account to accept this invitation.
          </p>
          <div className="flex flex-col gap-3">
            <Link to="/login">
              <Button className="w-full">Sign in</Button>
            </Link>
            <Link to="/register">
              <Button variant="secondary" className="w-full">Create account</Button>
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
      <div className="max-w-md w-full text-center">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
          Accept Invitation
        </h1>

        {error ? (
          <>
            <p className="text-red-600 dark:text-red-400 mb-6">{error}</p>
            <Link to={fallbackWorkspaceLink}>
              <Button variant="secondary">
                {workspaces && workspaces.length > 0 ? 'Go to Workspace' : 'Go to Login'}
              </Button>
            </Link>
          </>
        ) : (
          <>
            <p className="text-gray-600 dark:text-gray-400 mb-6">
              Click below to join the workspace.
            </p>
            <Button
              onClick={handleAccept}
              isLoading={acceptInvite.isPending}
              className="w-full"
            >
              Accept Invitation
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
