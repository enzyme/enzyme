import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Button, Input } from '../ui';
import { useAuth, useCreateWorkspace, useAcceptInvite } from '../../hooks';
import { ApiError } from '../../api';

export function RegisterForm() {
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [workspaceName, setWorkspaceName] = useState('');
  const [error, setError] = useState('');
  const { register, isRegistering } = useAuth();
  const createWorkspace = useCreateWorkspace();
  const acceptInvite = useAcceptInvite();
  const navigate = useNavigate();

  // Check if there's a pending invite - if so, don't show workspace name field
  const hasPendingInvite = !!sessionStorage.getItem('pendingInvite');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    if (!hasPendingInvite && !workspaceName.trim()) {
      setError('Workspace name is required');
      return;
    }

    try {
      await register({ email, password, display_name: displayName });

      // Check for pending invite and auto-accept it
      const pendingInvite = sessionStorage.getItem('pendingInvite');
      if (pendingInvite) {
        const { workspace } = await acceptInvite.mutateAsync(pendingInvite);
        sessionStorage.removeItem('pendingInvite');
        navigate(`/workspaces/${workspace.id}`, { replace: true });
      } else if (workspaceName.trim()) {
        // No pending invite - create a workspace and redirect to it
        const result = await createWorkspace.mutateAsync({ name: workspaceName.trim() });
        navigate(`/workspaces/${result.workspace.id}`, { replace: true });
      }
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('An error occurred. Please try again.');
      }
    }
  };

  const isSubmitting = isRegistering || createWorkspace.isPending || acceptInvite.isPending;

  return (
    <div className="w-full max-w-md">
      <div className="text-center mb-8">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
          Create an account
        </h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Get started with Feather
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="p-3 text-sm text-red-600 bg-red-50 dark:bg-red-900/20 dark:text-red-400 rounded-md">
            {error}
          </div>
        )}

        <Input
          type="text"
          label="Display name"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          placeholder="Your name"
          isRequired
          autoComplete="name"
        />

        <Input
          type="email"
          label="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          isRequired
          autoComplete="email"
        />

        <Input
          type="password"
          label="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="At least 8 characters"
          isRequired
          autoComplete="new-password"
        />

        <Input
          type="password"
          label="Confirm password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          placeholder="Confirm your password"
          isRequired
          autoComplete="new-password"
        />

        {!hasPendingInvite && (
          <Input
            type="text"
            label="Workspace name"
            value={workspaceName}
            onChange={(e) => setWorkspaceName(e.target.value)}
            placeholder="My Workspace"
            isRequired
          />
        )}

        <Button
          type="submit"
          className="w-full"
          isLoading={isSubmitting}
        >
          Create account
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-gray-600 dark:text-gray-400">
        Already have an account?{' '}
        <Link
          to="/login"
          className="text-primary-600 hover:text-primary-700 font-medium"
        >
          Sign in
        </Link>
      </p>
    </div>
  );
}
