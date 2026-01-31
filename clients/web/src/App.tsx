import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './lib/queryClient';
import { RequireAuth } from './components/auth';
import { AppLayout } from './components/layout';
import { Toaster } from './components/ui';
import {
  LoginPage,
  RegisterPage,
  WorkspaceListPage,
  WorkspaceLandingPage,
  WorkspaceSettingsPage,
  ChannelPage,
  AcceptInvitePage,
  InvitePage,
  ServerSettingsPage,
} from './pages';
import { useUIStore } from './stores/uiStore';

function DarkModeInitializer() {
  const { darkMode, setDarkMode } = useUIStore();

  useEffect(() => {
    // Check system preference on mount
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    setDarkMode(mediaQuery.matches);

    // Listen for changes
    const handler = (e: MediaQueryListEvent) => setDarkMode(e.matches);
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [setDarkMode]);

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode);
  }, [darkMode]);

  return null;
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <DarkModeInitializer />
        <Routes>
          {/* Public routes */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/invites/:code" element={<AcceptInvitePage />} />

          {/* Protected routes */}
          <Route
            path="/workspaces"
            element={
              <RequireAuth>
                <WorkspaceListPage />
              </RequireAuth>
            }
          />

          <Route
            path="/workspaces/:workspaceId"
            element={
              <RequireAuth>
                <AppLayout />
              </RequireAuth>
            }
          >
            <Route index element={<WorkspaceLandingPage />} />
            <Route path="channels/:channelId" element={<ChannelPage />} />
            <Route path="settings" element={<WorkspaceSettingsPage />} />
            <Route path="invite" element={<InvitePage />} />
          </Route>

          {/* Server settings */}
          <Route
            path="/settings"
            element={
              <RequireAuth>
                <ServerSettingsPage />
              </RequireAuth>
            }
          />

          {/* Redirect root to login */}
          <Route path="/" element={<Navigate to="/login" replace />} />

          {/* 404 */}
          <Route
            path="*"
            element={
              <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
                <div className="text-center">
                  <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
                    404
                  </h1>
                  <p className="text-gray-600 dark:text-gray-400">Page not found</p>
                </div>
              </div>
            }
          />
        </Routes>
        <Toaster />
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
