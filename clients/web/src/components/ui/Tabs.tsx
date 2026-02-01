import {
  Tabs as AriaTabs,
  TabList as AriaTabList,
  Tab as AriaTab,
  TabPanel as AriaTabPanel,
  type TabsProps as AriaTabsProps,
  type TabProps as AriaTabProps,
  type TabListProps as AriaTabListProps,
  type TabPanelProps as AriaTabPanelProps,
} from 'react-aria-components';
import { tv } from 'tailwind-variants';

const tabs = tv({
  slots: {
    root: '',
    list: 'flex border-b border-gray-200 dark:border-gray-700',
    tab: [
      'px-4 py-2 text-sm font-medium border-b-2 -mb-px cursor-pointer outline-none',
      'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300',
      'focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-inset',
      'data-[selected]:border-primary-600 data-[selected]:text-primary-600',
    ],
    panel: 'outline-none',
  },
});

interface TabsProps extends Omit<AriaTabsProps, 'className'> {
  className?: string;
}

export function Tabs({ className, ...props }: TabsProps) {
  const styles = tabs();
  return <AriaTabs className={styles.root({ className })} {...props} />;
}

interface TabListProps<T extends object> extends Omit<AriaTabListProps<T>, 'className'> {
  className?: string;
}

export function TabList<T extends object>({ className, ...props }: TabListProps<T>) {
  const styles = tabs();
  return <AriaTabList className={styles.list({ className })} {...props} />;
}

interface TabProps extends Omit<AriaTabProps, 'className'> {
  className?: string;
}

export function Tab({ className, ...props }: TabProps) {
  const styles = tabs();
  return <AriaTab className={styles.tab({ className })} {...props} />;
}

interface TabPanelProps extends Omit<AriaTabPanelProps, 'className'> {
  className?: string;
}

export function TabPanel({ className, ...props }: TabPanelProps) {
  const styles = tabs();
  return <AriaTabPanel className={styles.panel({ className })} {...props} />;
}
