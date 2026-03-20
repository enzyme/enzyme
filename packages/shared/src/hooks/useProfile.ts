import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { usersApi, type UpdateProfileInput } from '@enzyme/api-client';
import { authKeys, userKeys, messageKeys, threadKeys } from '../queryKeys';

export function useUserProfile(userId: string | null) {
  return useQuery({
    queryKey: userKeys.detail(userId!),
    queryFn: () => usersApi.getUser(userId!),
    enabled: !!userId,
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: UpdateProfileInput) => usersApi.updateProfile(input),
    onSuccess: (data) => {
      // Invalidate auth cache to update current user
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
      // Invalidate the user profile cache
      if (data.user) {
        queryClient.invalidateQueries({ queryKey: userKeys.detail(data.user.id) });
      }
      // Invalidate messages cache so updated display name appears
      queryClient.invalidateQueries({ queryKey: messageKeys.all });
      // Also invalidate thread messages
      queryClient.invalidateQueries({ queryKey: threadKeys.all });
    },
  });
}

export function useUploadAvatar() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (file: File) => usersApi.uploadAvatar(file),
    onSuccess: () => {
      // Invalidate auth cache to update current user's avatar
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
      // Invalidate user profile caches
      queryClient.invalidateQueries({ queryKey: userKeys.all });
      // Invalidate messages cache so updated avatar appears
      queryClient.invalidateQueries({ queryKey: messageKeys.all });
      // Also invalidate thread messages
      queryClient.invalidateQueries({ queryKey: threadKeys.all });
    },
  });
}

export function useDeleteAvatar() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => usersApi.deleteAvatar(),
    onSuccess: () => {
      // Invalidate auth cache to update current user's avatar
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
      // Invalidate user profile caches
      queryClient.invalidateQueries({ queryKey: userKeys.all });
      // Invalidate messages cache so updated avatar appears
      queryClient.invalidateQueries({ queryKey: messageKeys.all });
      // Also invalidate thread messages
      queryClient.invalidateQueries({ queryKey: threadKeys.all });
    },
  });
}
