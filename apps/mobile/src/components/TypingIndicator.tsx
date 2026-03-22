import { useEffect, useRef } from 'react';
import { View, Text, Animated } from 'react-native';
import { useTypingUsers } from '@enzyme/shared';

interface TypingIndicatorProps {
  channelId: string;
}

export function TypingIndicator({ channelId }: TypingIndicatorProps) {
  const typingUsers = useTypingUsers(channelId);

  if (typingUsers.length === 0) return null;

  let label: string;
  if (typingUsers.length === 1) {
    label = `${typingUsers[0].displayName} is typing`;
  } else if (typingUsers.length === 2) {
    label = `${typingUsers[0].displayName} and ${typingUsers[1].displayName} are typing`;
  } else {
    label = 'Several people are typing';
  }

  return (
    <View className="flex-row items-center px-4 py-1">
      <AnimatedDots />
      <Text className="ml-1.5 text-xs text-neutral-500 dark:text-neutral-400">{label}</Text>
    </View>
  );
}

function AnimatedDots() {
  const dot1 = useRef(new Animated.Value(0.3)).current;
  const dot2 = useRef(new Animated.Value(0.3)).current;
  const dot3 = useRef(new Animated.Value(0.3)).current;

  useEffect(() => {
    const animate = (dot: Animated.Value, delay: number) =>
      Animated.loop(
        Animated.sequence([
          Animated.delay(delay),
          Animated.timing(dot, { toValue: 1, duration: 300, useNativeDriver: true }),
          Animated.timing(dot, { toValue: 0.3, duration: 300, useNativeDriver: true }),
        ]),
      );

    const a1 = animate(dot1, 0);
    const a2 = animate(dot2, 200);
    const a3 = animate(dot3, 400);

    a1.start();
    a2.start();
    a3.start();

    return () => {
      a1.stop();
      a2.stop();
      a3.stop();
    };
  }, [dot1, dot2, dot3]);

  return (
    <View className="flex-row items-center" style={{ gap: 2 }}>
      {[dot1, dot2, dot3].map((dot, i) => (
        <Animated.View
          key={i}
          className="h-1.5 w-1.5 rounded-full bg-neutral-400 dark:bg-neutral-500"
          style={{ opacity: dot }}
        />
      ))}
    </View>
  );
}
