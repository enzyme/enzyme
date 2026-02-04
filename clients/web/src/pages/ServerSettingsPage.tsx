import { Link } from 'react-router-dom';
import { useAuth } from '../hooks';

export function ServerSettingsPage() {
  const { workspaces } = useAuth();

  // Get close link - navigate to first workspace or login
  const closeLink = workspaces && workspaces.length > 0
    ? `/workspaces/${workspaces[0].id}`
    : '/login';

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-4xl mx-auto p-8">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Server Settings
          </h1>
          <Link
            to={closeLink}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-medium text-gray-900 dark:text-white mb-1">
              Server Information
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              General information about your Feather server.
            </p>
          </div>

          <div className="p-6 space-y-4">
            <div className="flex justify-between items-center py-3 border-b border-gray-100 dark:border-gray-700">
              <div>
                <p className="font-medium text-gray-900 dark:text-white">Version</p>
                <p className="text-sm text-gray-500 dark:text-gray-400">Feather server version</p>
              </div>
              <span className="text-gray-600 dark:text-gray-300">1.0.0</span>
            </div>

            <div className="flex justify-between items-center py-3 border-b border-gray-100 dark:border-gray-700">
              <div>
                <p className="font-medium text-gray-900 dark:text-white">Database</p>
                <p className="text-sm text-gray-500 dark:text-gray-400">Database backend</p>
              </div>
              <span className="text-gray-600 dark:text-gray-300">SQLite</span>
            </div>

            <div className="flex justify-between items-center py-3">
              <div>
                <p className="font-medium text-gray-900 dark:text-white">Email</p>
                <p className="text-sm text-gray-500 dark:text-gray-400">Email service status</p>
              </div>
              <span className="text-yellow-600 dark:text-yellow-400">Not configured</span>
            </div>
          </div>
        </div>

        <div className="mt-6 bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-medium text-gray-900 dark:text-white mb-1">
              Administration
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Server administration options.
            </p>
          </div>

          <div className="p-6">
            <p className="text-gray-500 dark:text-gray-400 text-sm">
              Server configuration is managed through environment variables and the config file.
              See the documentation for more details.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
