import { UNSTABLE_ToastQueue as ToastQueue } from 'react-aria-components';

export interface ToastContent {
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
}

export const toastQueue = new ToastQueue<ToastContent>({
  maxVisibleToasts: 5,
});

export function toast(message: string, type: ToastContent['type'] = 'info') {
  toastQueue.add({ message, type }, { timeout: 5000 });
}
