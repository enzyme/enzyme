import { Pressable, Text } from 'react-native';

interface ActionButtonProps {
  label: string;
  onPress: () => void;
  destructive?: boolean;
}

export function ActionButton({ label, onPress, destructive = false }: ActionButtonProps) {
  return (
    <Pressable
      className="px-6 py-3.5 active:bg-neutral-100 dark:active:bg-neutral-700"
      onPress={onPress}
    >
      <Text
        className={`text-base ${
          destructive ? 'font-semibold text-red-500' : 'text-neutral-900 dark:text-white'
        }`}
      >
        {label}
      </Text>
    </Pressable>
  );
}
