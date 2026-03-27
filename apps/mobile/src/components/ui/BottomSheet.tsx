import type { ReactNode } from 'react';
import { View, Pressable, Modal } from 'react-native';

interface BottomSheetProps {
  visible: boolean;
  onDismiss: () => void;
  children: ReactNode;
  className?: string;
}

export function BottomSheet({ visible, onDismiss, children, className }: BottomSheetProps) {
  return (
    <Modal visible={visible} animationType="fade" transparent onRequestClose={onDismiss}>
      <Pressable className="flex-1 justify-end bg-black/40" onPress={onDismiss}>
        <Pressable className={`rounded-t-2xl bg-white pb-8 dark:bg-neutral-800 ${className ?? ''}`}>
          <View className="items-center py-2">
            <View className="h-1 w-10 rounded-full bg-neutral-300 dark:bg-neutral-600" />
          </View>
          {children}
        </Pressable>
      </Pressable>
    </Modal>
  );
}
