import { View, Text } from 'react-native';
import { formatDate } from '@enzyme/shared';

interface DateSeparatorProps {
  date: string;
}

export function DateSeparator({ date }: DateSeparatorProps) {
  return (
    <View className="my-3 flex-row items-center px-4">
      <View className="h-px flex-1 bg-neutral-200 dark:bg-neutral-700" />
      <Text className="mx-3 text-xs font-medium text-neutral-500 dark:text-neutral-400">
        {formatDate(date)}
      </Text>
      <View className="h-px flex-1 bg-neutral-200 dark:bg-neutral-700" />
    </View>
  );
}
