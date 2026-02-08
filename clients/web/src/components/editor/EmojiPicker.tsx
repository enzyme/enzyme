import { EmojiGrid } from '../ui';

interface EmojiPickerProps {
  onSelect: (emoji: string) => void;
}

export function EmojiPicker({ onSelect }: EmojiPickerProps) {
  return <EmojiGrid onSelect={onSelect} autoFocus />;
}
