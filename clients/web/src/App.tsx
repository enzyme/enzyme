import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './lib/queryClient';
import { RequireAuth } from './components/auth';
import { AppLayout } from './components/layout';
import { Toaster } from './components/ui';
import {
  LoginPage,
  RegisterPage,
  WorkspaceLandingPage,
  ChannelPage,
} from './pages';
import { useDarkMode } from './hooks/useDarkMode';

// Lazy-loaded pages (not on critical path)
const WorkspaceSettingsPage = lazy(() =>
  import('./pages/WorkspaceSettingsPage').then((m) => ({ default: m.WorkspaceSettingsPage }))
);
const InvitePage = lazy(() =>
  import('./pages/InvitePage').then((m) => ({ default: m.InvitePage }))
);
const ServerSettingsPage = lazy(() =>
  import('./pages/ServerSettingsPage').then((m) => ({ default: m.ServerSettingsPage }))
);
const AcceptInvitePage = lazy(() =>
  import('./pages/AcceptInvitePage').then((m) => ({ default: m.AcceptInvitePage }))
);
const AllUnreadsPage = lazy(() =>
  import('./pages/AllUnreadsPage').then((m) => ({ default: m.AllUnreadsPage }))
);
const ThreadsPage = lazy(() =>
  import('./pages/ThreadsPage').then((m) => ({ default: m.ThreadsPage }))
);

function PageSpinner() {
  return (
    <div className="flex items-center justify-center h-full">
      <div className="w-8 h-8 border-4 border-primary-200 border-t-primary-600 rounded-full animate-spin" />
    </div>
  );
}

function DarkModeInitializer() {
  // Hook handles localStorage persistence, system preference fallback, and DOM updates
  useDarkMode();
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
          <Route path="/invites/:code" element={<Suspense fallback={<PageSpinner />}><AcceptInvitePage /></Suspense>} />

          {/* Protected routes */}
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
            <Route path="unreads" element={<Suspense fallback={<PageSpinner />}><AllUnreadsPage /></Suspense>} />
            <Route path="threads" element={<Suspense fallback={<PageSpinner />}><ThreadsPage /></Suspense>} />
            <Route path="settings" element={<Suspense fallback={<PageSpinner />}><WorkspaceSettingsPage /></Suspense>} />
            <Route path="invite" element={<Suspense fallback={<PageSpinner />}><InvitePage /></Suspense>} />
          </Route>

          {/* Server settings */}
          <Route
            path="/settings"
            element={
              <RequireAuth>
                <Suspense fallback={<PageSpinner />}>
                  <ServerSettingsPage />
                </Suspense>
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
