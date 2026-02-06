import {
  Button,
  UNSTABLE_Toast as AriaToast,
  UNSTABLE_ToastContent as ToastContent,
  UNSTABLE_ToastRegion as ToastRegion,
} from 'react-aria-components';
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  InformationCircleIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { toastQueue, type ToastContent as ToastData } from './toast-store';

const typeIcons = {
  success: CheckCircleIcon,
  warning: ExclamationCircleIcon,
  error: XCircleIcon,
  info: InformationCircleIcon,
};

const typeIconColors = {
  success: 'text-green-400 dark:text-green-600',
  warning: 'text-yellow-400 dark:text-yellow-600',
  error: 'text-red-400 dark:text-red-600',
  info: 'text-blue-400 dark:text-blue-600',
};

export function Toaster() {
  return (
    <ToastRegion
      queue={toastQueue}
      className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 outline-none"
    >
      {({ toast }) => {
        const Icon = typeIcons[toast.content.type];
        return (
          <AriaToast<ToastData>
            toast={toast}
            className={({ isFocusVisible }) =>
              `flex items-center gap-2 px-4 py-3 rounded-lg shadow-lg bg-gray-900 text-white dark:bg-white dark:text-gray-900 animate-in slide-in-from-right fade-in duration-200${isFocusVisible ? ' ring-2 ring-white ring-offset-2 ring-offset-transparent' : ''}`
            }
          >
            <Icon className={`w-5 h-5 shrink-0 ${typeIconColors[toast.content.type]}`} />
            <ToastContent className="flex-1">
              <span>{toast.content.message}</span>
            </ToastContent>
            <Button
              slot="close"
              className="ml-2 hover:opacity-80 outline-none rounded focus-visible:ring-2 focus-visible:ring-white dark:text-gray-900"
            >
              <XMarkIcon className="w-4 h-4" />
            </Button>
          </AriaToast>
        );
      }}
    </ToastRegion>
  );
}
