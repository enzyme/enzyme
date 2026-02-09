import { cn } from '../../lib/utils';

const sizeClasses = {
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-7 h-7',
  xl: 'w-10 h-10',
};

interface CustomEmojiImgProps {
  name: string;
  url: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
}

export function CustomEmojiImg({ name, url, size = 'sm', className }: CustomEmojiImgProps) {
  return (
    <img
      src={url}
      alt={`:${name}:`}
      title={`:${name}:`}
      className={cn('inline-block align-text-bottom object-contain', sizeClasses[size], className)}
    />
  );
}
