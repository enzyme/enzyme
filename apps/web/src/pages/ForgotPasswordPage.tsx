import { Link } from 'react-router-dom';
import { ForgotPasswordForm } from '../components/auth';
import { usePageTitle, useServerInfo } from '../hooks';

export function ForgotPasswordPage() {
  usePageTitle('Forgot password');
  const { emailEnabled } = useServerInfo();

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-900">
      {emailEnabled ? (
        <ForgotPasswordForm />
      ) : (
        <div className="w-full max-w-md text-center">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            Password reset unavailable
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            Email is not configured on this server. Contact your administrator to reset your
            password.
          </p>
          <p className="mt-6 text-sm text-gray-600 dark:text-gray-400">
            <Link to="/login" className="font-medium text-blue-600 hover:text-blue-700">
              Back to sign in
            </Link>
          </p>
        </div>
      )}
    </div>
  );
}
