import { useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authApi, type LoginInput, type RegisterInput } from '../api/auth';
import { ApiError, setAuthToken, type User, type WorkspaceSummary } from '@feather/api-client';

const TOKEN_KEY = 'feather_auth_token';

function loadToken(): void {
  const token = localStorage.getItem(TOKEN_KEY);
  setAuthToken(token);
}

function saveToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
  setAuthToken(token);
}

function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  setAuthToken(null);
}

// Load token from localStorage on module load
loadToken();

export function useAuth() {
  const queryClient = useQueryClient();

  const { data, error, isFetched } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: authApi.me,
    retry: false,
    staleTime: Infinity, // Never consider stale
    gcTime: Infinity, // Never garbage collect
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    enabled: !!localStorage.getItem(TOKEN_KEY),
  });

  // 401 error means not authenticated, not an error state
  const isAuthError = error instanceof ApiError && error.status === 401;

  // Clear token on 401 from /auth/me
  useEffect(() => {
    if (isAuthError) {
      clearToken();
    }
  }, [isAuthError]);

  const loginMutation = useMutation({
    mutationFn: (input: LoginInput) => authApi.login(input),
    onSuccess: (data) => {
      saveToken(data.token);
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });

  const registerMutation = useMutation({
    mutationFn: (input: RegisterInput) => authApi.register(input),
    onSuccess: (data) => {
      saveToken(data.token);
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });

  const logoutMutation = useMutation({
    mutationFn: authApi.logout,
    onSuccess: () => {
      clearToken();
      queryClient.clear();
    },
  });

  return {
    user: data?.user as User | undefined,
    workspaces: data?.workspaces as WorkspaceSummary[] | undefined,
    isLoading: !isFetched && !!localStorage.getItem(TOKEN_KEY),
    isAuthenticated: !!data?.user,
    error: isAuthError ? null : error,
    login: loginMutation.mutateAsync,
    register: registerMutation.mutateAsync,
    logout: logoutMutation.mutateAsync,
    isLoggingIn: loginMutation.isPending,
    isRegistering: registerMutation.isPending,
    isLoggingOut: logoutMutation.isPending,
    loginError: loginMutation.error,
    registerError: registerMutation.error,
  };
}
