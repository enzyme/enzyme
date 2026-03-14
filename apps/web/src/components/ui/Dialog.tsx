import { Dialog as AriaDialog, type DialogProps as AriaDialogProps } from 'react-aria-components';

export function Dialog({ className = 'outline-none', ...props }: AriaDialogProps) {
  return <AriaDialog className={className} {...props} />;
}
