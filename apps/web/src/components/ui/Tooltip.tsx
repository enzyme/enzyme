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
  isDisabled?: boolean;
}

export function Tooltip({ children, content, placement = 'top', delay = 300, isDisabled }: TooltipProps) {
  return (
    <TooltipTrigger delay={delay} isDisabled={isDisabled}>
      {children}
      <AriaTooltip
        placement={placement}
        offset={6}
        className="max-w-xs rounded bg-gray-900 px-2 py-1 text-sm text-white shadow-lg dark:bg-gray-100 dark:text-gray-900"
      >
        {content}
      </AriaTooltip>
    </TooltipTrigger>
  );
}
