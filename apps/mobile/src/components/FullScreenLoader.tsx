import { View, ActivityIndicator } from 'react-native';

export function FullScreenLoader() {
  return (
    <View className="flex-1 items-center justify-center bg-white dark:bg-neutral-900">
      <ActivityIndicator size="large" />
    </View>
  );
}
