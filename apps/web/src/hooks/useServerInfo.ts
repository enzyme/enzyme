import { useQuery } from '@tanstack/react-query';
import { serverApi } from '../api';

export function useServerInfo() {
  const { data } = useQuery({
    queryKey: ['server-info'],
    queryFn: serverApi.getServerInfo,
    staleTime: Infinity,
    gcTime: Infinity,
  });

  return {
    emailEnabled: data?.email_enabled ?? true,
    filesEnabled: data?.files_enabled ?? true,
  };
}
