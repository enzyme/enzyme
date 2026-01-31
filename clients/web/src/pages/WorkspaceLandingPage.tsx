import { useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useChannels } from '../hooks';
import { Spinner } from '../components/ui';

export function WorkspaceLandingPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const navigate = useNavigate();
  const { data, isLoading } = useChannels(workspaceId);

  const channels = data?.channels ?? [];

  useEffect(() => {
    if (!isLoading && channels.length > 0) {
      // Redirect to first channel (prefer public channels)
      const publicChannel = channels.find((c) => c.type === 'public');
      const firstChannel = publicChannel || channels[0];
      navigate(`/workspaces/${workspaceId}/channels/${firstChannel.id}`, { replace: true });
    }
  }, [channels, isLoading, workspaceId, navigate]);

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (channels.length === 0) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-gray-500 dark:text-gray-400">
        <svg className="w-16 h-16 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14"
          />
        </svg>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
          No channels yet
        </h2>
        <p className="text-sm">Create a channel to start messaging</p>
      </div>
    );
  }

  return (
    <div className="flex-1 flex items-center justify-center">
      <Spinner size="lg" />
    </div>
  );
}
