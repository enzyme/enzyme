import { View, Text, Image } from 'react-native';
import { getInitials, getAvatarColor } from '@enzyme/shared';

const SIZES = {
  sm: { container: 28, text: 12 },
  md: { container: 36, text: 14 },
  lg: { container: 48, text: 18 },
} as const;

// NativeWind can't resolve dynamic Tailwind classes, so map to hex values.
const COLOR_MAP: Record<string, string> = {
  'bg-red-500': '#ef4444',
  'bg-orange-500': '#f97316',
  'bg-amber-500': '#f59e0b',
  'bg-yellow-500': '#eab308',
  'bg-lime-500': '#84cc16',
  'bg-green-500': '#22c55e',
  'bg-emerald-500': '#10b981',
  'bg-teal-500': '#14b8a6',
  'bg-cyan-500': '#06b6d4',
  'bg-sky-500': '#0ea5e9',
  'bg-blue-500': '#3b82f6',
  'bg-violet-500': '#8b5cf6',
  'bg-purple-500': '#a855f7',
  'bg-fuchsia-500': '#d946ef',
  'bg-pink-500': '#ec4899',
  'bg-rose-500': '#f43f5e',
};

interface AvatarProps {
  user: { display_name: string; avatar_url?: string | null; id?: string };
  size: 'sm' | 'md' | 'lg';
}

export function Avatar({ user, size }: AvatarProps) {
  const { container, text } = SIZES[size];

  if (user.avatar_url) {
    return (
      <Image
        source={{ uri: user.avatar_url }}
        style={{ width: container, height: container, borderRadius: container / 2 }}
      />
    );
  }

  const colorClass = getAvatarColor(user.id ?? user.display_name);
  const bgColor = COLOR_MAP[colorClass] ?? '#6b7280';

  return (
    <View
      className="items-center justify-center rounded-full"
      style={{ width: container, height: container, backgroundColor: bgColor }}
    >
      <Text className="font-semibold text-white" style={{ fontSize: text }}>
        {getInitials(user.display_name)}
      </Text>
    </View>
  );
}
