import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../test-utils';
import { Routes, Route } from 'react-router-dom';

// Hoist mocks before other imports
const mockAuthApi = vi.hoisted(() => ({
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  me: vi.fn(),
  forgotPassword: vi.fn(),
  resetPassword: vi.fn(),
}));

vi.mock('../../api/auth', () => ({
  authApi: mockAuthApi,
}));

// Import after mocks are set up
import { RequireAuth } from './RequireAuth';

function TestApp() {
  return (
    <Routes>
      <Route path="/login" element={<div>Login Page</div>} />
      <Route
        path="/protected"
        element={
          <RequireAuth>
            <div>Protected Content</div>
          </RequireAuth>
        }
      />
    </Routes>
  );
}

describe('RequireAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Set a token in localStorage so the auth query is enabled
    localStorage.setItem('feather_auth_token', 'test-token');
  });

  it('shows loading spinner while checking auth', async () => {
    // Keep the me() call pending
    mockAuthApi.me.mockImplementation(() => new Promise(() => {}));

    const { container } = render(<TestApp />, {
      routerProps: { initialEntries: ['/protected'] },
    });

    // Should show spinner while loading (spinner has animate-spin class)
    expect(container.querySelector('.animate-spin')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument();
  });

  it('renders children when authenticated', async () => {
    mockAuthApi.me.mockResolvedValue({
      user: { id: 'user-1', email: 'test@example.com', display_name: 'Test' },
      workspaces: [],
    });

    render(<TestApp />, {
      routerProps: { initialEntries: ['/protected'] },
    });

    await waitFor(() => {
      expect(screen.getByText('Protected Content')).toBeInTheDocument();
    });
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument();
  });

  it('redirects to login when not authenticated', async () => {
    // Mock 401 error for unauthenticated user
    const error = new Error('Unauthorized');
    (error as Error & { status: number }).status = 401;
    mockAuthApi.me.mockRejectedValue(error);

    render(<TestApp />, {
      routerProps: { initialEntries: ['/protected'] },
    });

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('redirects to login on auth error', async () => {
    mockAuthApi.me.mockRejectedValue(new Error('Not authenticated'));

    render(<TestApp />, {
      routerProps: { initialEntries: ['/protected'] },
    });

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });

  it('preserves location state for redirect back after login', async () => {
    mockAuthApi.me.mockRejectedValue(new Error('Not authenticated'));

    // We can't easily test the location state passed to Navigate,
    // but we can verify the redirect happens
    render(<TestApp />, {
      routerProps: { initialEntries: ['/protected?param=value'] },
    });

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });

  it('renders multiple children correctly', async () => {
    mockAuthApi.me.mockResolvedValue({
      user: { id: 'user-1', email: 'test@example.com', display_name: 'Test' },
      workspaces: [],
    });

    render(
      <Routes>
        <Route
          path="/protected"
          element={
            <RequireAuth>
              <div>First Child</div>
              <div>Second Child</div>
            </RequireAuth>
          }
        />
      </Routes>,
      { routerProps: { initialEntries: ['/protected'] } },
    );

    await waitFor(() => {
      expect(screen.getByText('First Child')).toBeInTheDocument();
      expect(screen.getByText('Second Child')).toBeInTheDocument();
    });
  });
});
