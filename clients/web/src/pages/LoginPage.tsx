import { Navigate } from 'react-router-dom';
import { LoginForm } from '../components/auth';
import { useAuth } from '../hooks';
import { Spinner } from '../components/ui';

export function LoginPage() {
  const { isAuthenticated, isLoading, workspaces } = useAuth();

  if (isAuthenticated) {
    // Check for pending invite
    const pendingInvite = sessionStorage.getItem('pendingInvite');
    if (pendingInvite) {
      // Don't remove here - StrictMode double-renders would clear it before redirect
      // AcceptInvitePage will clear it after processing
      return <Navigate to={`/invites/${pendingInvite}`} replace />;
    }
    // Redirect to first workspace or workspace list
    if (workspaces && workspaces.length > 0) {
      return <Navigate to={`/workspaces/${workspaces[0].id}`} replace />;
    }
    return <Navigate to="/workspaces" replace />;
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
