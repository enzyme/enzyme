import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, userEvent } from '../../test-utils';

// Hoist all mocks before other imports
const mockLogin = vi.hoisted(() => vi.fn());
const mockAuthApi = vi.hoisted(() => ({
  login: mockLogin,
  register: vi.fn(),
  logout: vi.fn(),
  me: vi.fn(),
  forgotPassword: vi.fn(),
  resetPassword: vi.fn(),
}));

const MockApiError = vi.hoisted(() => {
  return class MockApiError extends Error {
    status: number;
    constructor(message: string, status: number = 400) {
      super(message);
      this.status = status;
    }
  };
});

vi.mock('../../api/auth', () => ({
  authApi: mockAuthApi,
}));

vi.mock('../../api', () => ({
  ApiError: MockApiError,
  serverApi: {
    getServerInfo: vi.fn().mockResolvedValue({ version: '0.0.0', email_enabled: true }),
  },
}));

// Import after mocks are set up
import { LoginForm } from './LoginForm';

describe('LoginForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default: user is not authenticated
    mockAuthApi.me.mockRejectedValue(new Error('Not authenticated'));
  });

  it('renders login form with email and password fields', () => {
    render(<LoginForm />);

    expect(screen.getByText('Welcome back')).toBeInTheDocument();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
  });

  it('renders link to register page', () => {
    render(<LoginForm />);

    const signUpLink = screen.getByRole('link', { name: /sign up/i });
    expect(signUpLink).toBeInTheDocument();
    expect(signUpLink).toHaveAttribute('href', '/register');
  });

  it('allows entering email and password', async () => {
    const user = userEvent.setup();
    render(<LoginForm />);

    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);

    await user.type(emailInput, 'test@example.com');
    await user.type(passwordInput, 'password123');

    expect(emailInput).toHaveValue('test@example.com');
    expect(passwordInput).toHaveValue('password123');
  });

  it('calls login on form submission', async () => {
    mockLogin.mockResolvedValue({ user: { id: 'user-1' } });
    const user = userEvent.setup();
    render(<LoginForm />);

    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    await user.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });
  });

  it('shows error message on login failure', async () => {
    mockLogin.mockRejectedValue(new MockApiError('Invalid email or password', 401));
    const user = userEvent.setup();
    render(<LoginForm />);

    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/password/i), 'wrongpassword');
    await user.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByText('Invalid email or password')).toBeInTheDocument();
    });
  });

  it('shows generic error for non-API errors', async () => {
    mockLogin.mockRejectedValue(new Error('Network error'));
    const user = userEvent.setup();
    render(<LoginForm />);

    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    await user.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByText('An error occurred. Please try again.')).toBeInTheDocument();
    });
  });

  it('disables submit button while logging in', async () => {
    // Create a promise that we can control
    let resolveLogin: (value: { user: { id: string } }) => void;
    mockLogin.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveLogin = resolve;
        }),
    );
    const user = userEvent.setup();
    render(<LoginForm />);

    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    await user.click(screen.getByRole('button', { name: /sign in/i }));

    // Button should be disabled while logging in
    await waitFor(() => {
      expect(screen.getByRole('button')).toBeDisabled();
    });

    // Resolve the login
    resolveLogin!({ user: { id: 'user-1' } });

    await waitFor(() => {
      expect(screen.getByRole('button')).not.toBeDisabled();
    });
  });
});
