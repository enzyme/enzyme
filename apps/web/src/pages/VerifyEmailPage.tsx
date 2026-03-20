import { useRef, useEffect } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { usePageTitle } from '../hooks';
import { authApi, ApiError } from '@enzyme/api-client';
import { authKeys } from '@enzyme/shared';

function CenteredLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-900">
      <div className="w-full max-w-md">{children}</div>
    </div>
  );
}

export function VerifyEmailPage() {
  usePageTitle('Verify email');

  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');
  const queryClient = useQueryClient();
  const hasFired = useRef(false);

  const verifyEmail = useMutation({
    mutationFn: (t: string) => authApi.verifyEmail(t),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });

  useEffect(() => {
    if (token && !hasFired.current) {
      hasFired.current = true;
      verifyEmail.mutate(token);
    }
  }, [token, verifyEmail]);

  if (!token) {
    return (
      <CenteredLayout>
        <div className="mb-8 text-center">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            Invalid verification link
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            This email verification link is invalid.
          </p>
        </div>
        <p className="text-center text-sm text-gray-600 dark:text-gray-400">
          <Link to="/login" className="font-medium text-blue-600 hover:text-blue-700">
            Go to login
          </Link>
        </p>
      </CenteredLayout>
    );
  }

  if (verifyEmail.isIdle || verifyEmail.isPending) {
    return (
      <CenteredLayout>
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Verifying email...</h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">Please wait a moment.</p>
        </div>
      </CenteredLayout>
    );
  }

  if (verifyEmail.isSuccess) {
    return (
      <CenteredLayout>
        <div className="mb-8 text-center">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Email verified!</h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            Your email address has been verified successfully.
          </p>
        </div>
        <p className="text-center text-sm text-gray-600 dark:text-gray-400">
          <Link to="/login" className="font-medium text-blue-600 hover:text-blue-700">
            Go to login
          </Link>
        </p>
      </CenteredLayout>
    );
  }

  const errorMessage =
    verifyEmail.error instanceof ApiError
      ? verifyEmail.error.message
      : 'An error occurred. Please try again.';

  return (
    <CenteredLayout>
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Verification failed</h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">{errorMessage}</p>
      </div>
      <p className="text-center text-sm text-gray-600 dark:text-gray-400">
        <Link to="/login" className="font-medium text-blue-600 hover:text-blue-700">
          Go to login
        </Link>
      </p>
    </CenteredLayout>
  );
}
