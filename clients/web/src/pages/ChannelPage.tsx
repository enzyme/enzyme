import { useParams } from 'react-router-dom';
import { useChannels } from '../hooks';
import { MessageList, MessageComposer } from '../components/message';
import { Spinner } from '../components/ui';
import { getChannelIcon } from '../lib/utils';

export function ChannelPage() {
  const { workspaceId, channelId } = useParams<{
    workspaceId: string;
    channelId: string;
  }>();

  const { data: channelsData, isLoading } = useChannels(workspaceId);
  const channel = channelsData?.channels.find((c) => c.id === channelId);

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!channelId || !workspaceId) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500 dark:text-gray-400">
        Select a channel to start messaging
      </div>
    );
  }

  if (!channel) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500 dark:text-gray-400">
        Channel not found
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Channel header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
        <div className="flex items-center gap-2">
          <span className="text-gray-500 dark:text-gray-400">
            {getChannelIcon(channel.type)}
          </span>
          <h1 className="font-semibold text-gray-900 dark:text-white">
            {channel.name}
          </h1>
        </div>
        {channel.description && (
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {channel.description}
          </p>
        )}
      </div>

      {/* Messages */}
      <MessageList channelId={channelId} />

      {/* Composer */}
      <MessageComposer
        channelId={channelId}
        workspaceId={workspaceId}
        placeholder={`Message ${getChannelIcon(channel.type)}${channel.name}`}
      />
    </div>
  );
}
