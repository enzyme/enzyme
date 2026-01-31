import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authApi, type LoginInput, type RegisterInput } from '../api/auth';
import { ApiError, type User, type WorkspaceSummary } from '@feather/api-client';

export function useAuth() {
  const queryClient = useQueryClient();

  const { data, isLoading, error, isFetched } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: authApi.me,
    retry: false,
    staleTime: Infinity, // Never consider stale
    gcTime: Infinity, // Never garbage collect
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  // 401 error means not authenticated, not an error state
  const isAuthError = error instanceof ApiError && error.status === 401;

  const loginMutation = useMutation({
    mutationFn: (input: LoginInput) => authApi.login(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });

  const registerMutation = useMutation({
    mutationFn: (input: RegisterInput) => authApi.register(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });

  const logoutMutation = useMutation({
    mutationFn: authApi.logout,
    onSuccess: () => {
      queryClient.clear();
    },
  });

  return {
    user: data?.user as User | undefined,
    workspaces: data?.workspaces as WorkspaceSummary[] | undefined,
    isLoading: !isFetched,
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
