import { type ReactNode } from 'react';
import {
  Dialog,
  DialogTrigger,
  Modal as AriaModal,
  ModalOverlay,
  Heading,
} from 'react-aria-components';
import { tv, type VariantProps } from 'tailwind-variants';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { IconButton } from './IconButton';

const modal = tv({
  slots: {
    overlay: [
      'fixed inset-0 z-50 flex items-center justify-center',
      'bg-black/50',
      'entering:animate-in entering:fade-in entering:duration-200',
      'exiting:animate-out exiting:fade-out exiting:duration-150',
    ],
    container: [
      'relative w-full mx-4 bg-white dark:bg-gray-800 rounded-lg shadow-xl',
      'entering:animate-in entering:zoom-in-95 entering:duration-200',
      'exiting:animate-out exiting:zoom-out-95 exiting:duration-150',
    ],
    header:
      'flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700',
    title: 'text-lg font-semibold text-gray-900 dark:text-white',
    content: 'p-6',
  },
  variants: {
    size: {
      sm: { container: 'max-w-sm' },
      md: { container: 'max-w-md' },
      lg: { container: 'max-w-lg' },
      xl: { container: 'max-w-2xl' },
      settings: {
        container: 'max-w-4xl h-[85vh] max-h-[720px] flex flex-col',
        header: 'pl-4 pr-3 py-3',
        title: 'text-base',
      },
    },
  },
  defaultVariants: {
    size: 'md',
  },
});

type ModalVariants = VariantProps<typeof modal>;

interface ModalProps extends ModalVariants {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
}

export function Modal({ isOpen, onClose, title, children, size }: ModalProps) {
  const styles = modal({ size });

  return (
    <ModalOverlay
      isOpen={isOpen}
      onOpenChange={(open) => !open && onClose()}
      isDismissable
      className={styles.overlay()}
    >
      <AriaModal className={styles.container()}>
        <Dialog
          className={size === 'settings' ? 'flex h-full flex-col outline-none' : 'outline-none'}
        >
          {({ close }) => (
            <>
              {title && (
                <div className={styles.header()}>
                  <Heading slot="title" className={styles.title()}>
                    {title}
                  </Heading>
                  <IconButton
                    onPress={close}
                    aria-label="Close"
                    size="sm"
                    className={size === 'settings' ? 'p-1.5' : undefined}
                  >
                    <XMarkIcon className={size === 'settings' ? 'h-4 w-4' : 'h-5 w-5'} />
                  </IconButton>
                </div>
              )}
              <div
                className={
                  size === 'settings' ? 'min-h-0 flex-1 overflow-hidden' : styles.content()
                }
              >
                {children}
              </div>
            </>
          )}
        </Dialog>
      </AriaModal>
    </ModalOverlay>
  );
}

export { DialogTrigger };
