import { type ReactNode } from 'react';
import { Button as AriaButton, type ButtonProps as AriaButtonProps } from 'react-aria-components';
import { tv, type VariantProps } from 'tailwind-variants';

const iconButton = tv({
  base: [
    'inline-flex items-center justify-center rounded transition-colors cursor-pointer',
    'focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-blue-500',
    'disabled:opacity-50 disabled:cursor-not-allowed',
  ],
  variants: {
    variant: {
      ghost:
        'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-gray-200',
      danger:
        'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 hover:text-red-600 dark:hover:text-red-400',
    },
    size: {
      xs: 'p-0.5',
      sm: 'p-1',
      md: 'p-1.5',
    },
  },
  defaultVariants: {
    variant: 'ghost',
    size: 'md',
  },
});

type IconButtonVariants = VariantProps<typeof iconButton>;

interface IconButtonProps
  extends Omit<AriaButtonProps, 'className' | 'children'>, IconButtonVariants {
  'aria-label': string;
  className?: string;
  children: ReactNode;
}

export function IconButton({ className, variant, size, children, ...props }: IconButtonProps) {
  return (
    <AriaButton className={iconButton({ variant, size, className })} {...props}>
      {children}
    </AriaButton>
  );
}
