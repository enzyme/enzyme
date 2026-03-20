import { useQuery } from '@tanstack/react-query';
import { serverApi } from '@enzyme/api-client';
import { serverKeys } from '@enzyme/shared';

export function useServerInfo() {
  const { data } = useQuery({
    queryKey: serverKeys.info(),
    queryFn: serverApi.getServerInfo,
    staleTime: Infinity,
    gcTime: Infinity,
  });

  return {
    emailEnabled: data?.email_enabled ?? true,
    filesEnabled: data?.files_enabled ?? true,
  };
}
