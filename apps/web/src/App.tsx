import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './lib/queryClient';
import { RequireAuth } from './components/auth';
import { AppLayout } from './components/layout';
import { Toaster } from './components/ui';
import { LoginPage, RegisterPage, WorkspaceLandingPage, ChannelPage } from './pages';
import { useDarkMode } from './hooks/useDarkMode';

// Lazy-loaded pages (not on critical path)
const AcceptInvitePage = lazy(() =>
  import('./pages/AcceptInvitePage').then((m) => ({ default: m.AcceptInvitePage })),
);
const AllUnreadsPage = lazy(() =>
  import('./pages/AllUnreadsPage').then((m) => ({ default: m.AllUnreadsPage })),
);
const ThreadsPage = lazy(() =>
  import('./pages/ThreadsPage').then((m) => ({ default: m.ThreadsPage })),
);
const ScheduledMessagesPage = lazy(() =>
  import('./pages/ScheduledMessagesPage').then((m) => ({ default: m.ScheduledMessagesPage })),
);

function PageSpinner() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-blue-200 border-t-blue-600" />
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
          <Route
            path="/invites/:code"
            element={
              <Suspense fallback={<PageSpinner />}>
                <AcceptInvitePage />
              </Suspense>
            }
          />

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
            <Route
              path="unreads"
              element={
                <Suspense fallback={<PageSpinner />}>
                  <AllUnreadsPage />
                </Suspense>
              }
            />
            <Route
              path="threads"
              element={
                <Suspense fallback={<PageSpinner />}>
                  <ThreadsPage />
                </Suspense>
              }
            />
            <Route
              path="scheduled"
              element={
                <Suspense fallback={<PageSpinner />}>
                  <ScheduledMessagesPage />
                </Suspense>
              }
            />
          </Route>

          {/* Redirect root to login */}
          <Route path="/" element={<Navigate to="/login" replace />} />

          {/* 404 */}
          <Route
            path="*"
            element={
              <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
                <div className="text-center">
                  <h1 className="mb-4 text-4xl font-bold text-gray-900 dark:text-white">404</h1>
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
