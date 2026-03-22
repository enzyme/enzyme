import { View, Text } from 'react-native';

interface UnreadBadgeProps {
  count: number;
}

export function UnreadBadge({ count }: UnreadBadgeProps) {
  if (count === 0) return null;

  const display = count > 99 ? '99+' : String(count);

  return (
    <View className="min-w-5 items-center justify-center rounded-full bg-red-500 px-1.5 py-0.5">
      <Text className="text-xs font-bold text-white">{display}</Text>
    </View>
  );
}
