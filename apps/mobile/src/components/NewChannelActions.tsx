import { BottomSheet } from './ui/BottomSheet';
import { ActionButton } from './ui/ActionButton';

interface NewChannelActionsProps {
  visible: boolean;
  onDismiss: () => void;
  onCreateChannel: () => void;
  onBrowseChannels: () => void;
}

export function NewChannelActions({
  visible,
  onDismiss,
  onCreateChannel,
  onBrowseChannels,
}: NewChannelActionsProps) {
  return (
    <BottomSheet visible={visible} onDismiss={onDismiss}>
      <ActionButton
        label="Create Channel"
        onPress={() => {
          onDismiss();
          onCreateChannel();
        }}
      />
      <ActionButton
        label="Browse Channels"
        onPress={() => {
          onDismiss();
          onBrowseChannels();
        }}
      />
    </BottomSheet>
  );
}
