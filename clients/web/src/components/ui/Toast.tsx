import {
  Button,
  UNSTABLE_Toast as AriaToast,
  UNSTABLE_ToastContent as ToastContent,
  UNSTABLE_ToastRegion as ToastRegion,
} from 'react-aria-components';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { toastQueue, type ToastContent as ToastData } from './toast-store';

const typeStyles = {
  success: 'bg-green-600',
  error: 'bg-red-600',
  info: 'bg-gray-800',
};

export function Toaster() {
  return (
    <ToastRegion
      queue={toastQueue}
      className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 outline-none"
    >
      {({ toast }) => (
        <AriaToast<ToastData>
          toast={toast}
          className={({ isFocusVisible }) =>
            `flex items-center gap-2 px-4 py-3 rounded-lg shadow-lg text-white animate-in slide-in-from-right fade-in duration-200 ${typeStyles[toast.content.type]}${isFocusVisible ? ' ring-2 ring-white ring-offset-2 ring-offset-transparent' : ''}`
          }
        >
          <ToastContent className="flex-1">
            <span>{toast.content.message}</span>
          </ToastContent>
          <Button
            slot="close"
            className="ml-2 hover:opacity-80 outline-none rounded focus-visible:ring-2 focus-visible:ring-white"
          >
            <XMarkIcon className="w-4 h-4" />
          </Button>
        </AriaToast>
      )}
    </ToastRegion>
  );
}
