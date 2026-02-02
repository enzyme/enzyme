import { type ReactNode } from 'react';
import {
  Menu as AriaMenu,
  MenuItem as AriaMenuItem,
  MenuTrigger,
  Popover,
  SubmenuTrigger as AriaSubmenuTrigger,
  Section,
  Header,
  Separator as AriaSeparator,
  type MenuItemProps as AriaMenuItemProps,
} from 'react-aria-components';
import { tv, type VariantProps } from 'tailwind-variants';
import { ChevronRightIcon } from '@heroicons/react/24/outline';

const menu = tv({
  slots: {
    popover: [
      'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700',
      'rounded-lg shadow-lg py-1 min-w-[160px] z-50',
      'entering:animate-in entering:fade-in entering:zoom-in-95 entering:duration-150',
      'exiting:animate-out exiting:fade-out exiting:zoom-out-95 exiting:duration-100',
    ],
    menu: 'outline-none',
    item: [
      'w-full text-left px-3 py-1.5 text-sm flex items-center gap-2 whitespace-nowrap cursor-pointer',
      'outline-none',
      'focus:bg-gray-100 dark:focus:bg-gray-700',
      'disabled:opacity-50 disabled:cursor-not-allowed',
    ],
    section: '',
    header: 'px-3 py-2 border-b border-gray-200 dark:border-gray-700',
    separator: 'border-t border-gray-200 dark:border-gray-700 my-1',
  },
  variants: {
    variant: {
      default: {
        item: 'text-gray-700 dark:text-gray-200',
      },
      danger: {
        item: 'text-red-600 dark:text-red-400',
      },
    },
  },
  defaultVariants: {
    variant: 'default',
  },
});

type MenuVariants = VariantProps<typeof menu>;

interface MenuProps {
  trigger: ReactNode;
  children: ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  align?: 'start' | 'end';
  placement?: 'bottom' | 'top' | 'left' | 'right';
}

export function Menu({
  trigger,
  children,
  open,
  onOpenChange,
  align = 'end',
  placement = 'bottom',
}: MenuProps) {
  const styles = menu();

  const popoverPlacement = `${placement} ${align}` as const;

  return (
    <MenuTrigger isOpen={open} onOpenChange={onOpenChange}>
      {trigger}
      <Popover
        placement={popoverPlacement}
        className={styles.popover()}
      >
        <AriaMenu className={styles.menu()}>
          {children}
        </AriaMenu>
      </Popover>
    </MenuTrigger>
  );
}

interface MenuItemProps extends Omit<AriaMenuItemProps, 'className'>, MenuVariants {
  icon?: ReactNode;
  children: ReactNode;
}

export function MenuItem({
  children,
  variant,
  icon,
  ...props
}: MenuItemProps) {
  const styles = menu({ variant });

  return (
    <AriaMenuItem className={styles.item()} {...props}>
      {icon}
      {children}
    </AriaMenuItem>
  );
}

interface SubmenuTriggerProps extends MenuVariants {
  icon?: ReactNode;
  label: ReactNode;
  children: ReactNode;
}

export function SubmenuTrigger({
  label,
  children,
  variant,
  icon,
}: SubmenuTriggerProps) {
  const styles = menu({ variant });

  return (
    <AriaSubmenuTrigger>
      <AriaMenuItem className={styles.item()}>
        {icon}
        <span className="flex-1">{label}</span>
        <ChevronRightIcon className="w-4 h-4" />
      </AriaMenuItem>
      <Popover offset={-2} crossOffset={-4} className={styles.popover()}>
        <AriaMenu className={styles.menu()}>
          {children}
        </AriaMenu>
      </Popover>
    </AriaSubmenuTrigger>
  );
}

interface MenuSectionProps {
  children: ReactNode;
}

export function MenuSection({ children }: MenuSectionProps) {
  const styles = menu();

  return (
    <Section className={styles.section()}>
      {children}
    </Section>
  );
}

interface MenuHeaderProps {
  children: ReactNode;
}

export function MenuHeader({ children }: MenuHeaderProps) {
  const styles = menu();

  return (
    <Header className={styles.header()}>
      {children}
    </Header>
  );
}

export function MenuSeparator() {
  const styles = menu();

  return <AriaSeparator className={styles.separator()} />;
}

export { MenuTrigger };
