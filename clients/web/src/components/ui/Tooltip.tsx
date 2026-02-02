import { type ReactNode } from 'react';
import {
  TooltipTrigger,
  Tooltip as AriaTooltip,
  type TooltipProps as AriaTooltipProps,
} from 'react-aria-components';

interface TooltipProps {
  children: ReactNode;
  content: ReactNode;
  placement?: AriaTooltipProps['placement'];
  delay?: number;
}

export function Tooltip({
  children,
  content,
  placement = 'top',
  delay = 300,
}: TooltipProps) {
  return (
    <TooltipTrigger delay={delay}>
      {children}
      <AriaTooltip
        placement={placement}
        offset={6}
        className="bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 px-2 py-1 rounded text-sm shadow-lg max-w-xs"
      >
        {content}
      </AriaTooltip>
    </TooltipTrigger>
  );
}
