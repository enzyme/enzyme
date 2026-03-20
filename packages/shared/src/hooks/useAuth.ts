import { useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  authApi,
  ApiError,
  setAuthToken,
  getAuthToken,
  type LoginInput,
  type RegisterInput,
  type User,
  type WorkspaceSummary,
} from '@enzyme/api-client';
import { authKeys } from '../queryKeys';

/**
 * Shared auth hook. Consumers MUST call `setTokenStorage()` from
 * `@enzyme/api-client` before mounting any component that uses this hook,
 * otherwise `getAuthToken()` will always return `null` and the query
 * will remain permanently disabled (silently unauthenticated).
 */
export function useAuth() {
  const queryClient = useQueryClient();

  const { data, error, isFetched } = useQuery({
    queryKey: authKeys.me(),
    queryFn: authApi.me,
    retry: false,
    staleTime: Infinity, // Never consider stale
    gcTime: Infinity, // Never garbage collect
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    enabled: !!getAuthToken(),
  });

  // 401 error means not authenticated, not an error state
  const isAuthError = error instanceof ApiError && error.status === 401;

  // Clear token on 401 from /auth/me
  useEffect(() => {
    if (isAuthError) {
      setAuthToken(null);
    }
  }, [isAuthError]);

  const loginMutation = useMutation({
    mutationFn: (input: LoginInput) => authApi.login(input),
    onSuccess: (data) => {
      setAuthToken(data.token);
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });

  const registerMutation = useMutation({
    mutationFn: (input: RegisterInput) => authApi.register(input),
    onSuccess: (data) => {
      setAuthToken(data.token);
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });

  const logoutMutation = useMutation({
    mutationFn: authApi.logout,
    onSuccess: () => {
      setAuthToken(null);
      queryClient.clear();
    },
  });

  return {
    user: data?.user as User | undefined,
    workspaces: data?.workspaces as WorkspaceSummary[] | undefined,
    isLoading: !isFetched && !!getAuthToken(),
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
