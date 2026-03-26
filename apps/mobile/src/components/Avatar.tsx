import { View, Text, Image, useColorScheme } from 'react-native';
import { getInitials, getAvatarColor, useUserPresence } from '@enzyme/shared';

const SIZES = {
  sm: { container: 28, text: 12, dot: 8 },
  md: { container: 36, text: 14, dot: 10 },
  lg: { container: 48, text: 18, dot: 12 },
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

const PRESENCE_COLORS: Record<string, string> = {
  online: '#22c55e',
};

interface AvatarProps {
  user: {
    display_name: string;
    avatar_url?: string | null;
    gravatar_url?: string | null;
    id?: string;
  };
  size: 'sm' | 'md' | 'lg';
  showPresence?: boolean;
}

function PresenceDot({ userId, dotSize }: { userId: string; dotSize: number }) {
  const presence = useUserPresence(userId);
  const colorScheme = useColorScheme();
  const color = presence ? PRESENCE_COLORS[presence] : undefined;
  if (!color) return null;

  return (
    <View
      style={{
        position: 'absolute',
        bottom: -1,
        right: -1,
        width: dotSize,
        height: dotSize,
        borderRadius: dotSize / 2,
        backgroundColor: color,
        borderWidth: 2,
        borderColor: colorScheme === 'dark' ? '#171717' : '#ffffff',
      }}
    />
  );
}

export function Avatar({ user, size, showPresence }: AvatarProps) {
  const { container, text, dot } = SIZES[size];

  const imageUrl = user.avatar_url || user.gravatar_url;

  const avatar = imageUrl ? (
    <Image
      source={{ uri: imageUrl }}
      style={{ width: container, height: container, borderRadius: container / 2 }}
    />
  ) : (
    <View
      className="items-center justify-center rounded-full"
      style={{
        width: container,
        height: container,
        backgroundColor: COLOR_MAP[getAvatarColor(user.id ?? user.display_name)] ?? '#6b7280',
      }}
    >
      <Text className="font-semibold text-white" style={{ fontSize: text }}>
        {getInitials(user.display_name)}
      </Text>
    </View>
  );

  if (showPresence && user.id) {
    return (
      <View style={{ width: container, height: container }}>
        {avatar}
        <PresenceDot userId={user.id} dotSize={dot} />
      </View>
    );
  }

  return avatar;
}
