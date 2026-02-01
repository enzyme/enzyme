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
    header: 'flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700',
    title: 'text-lg font-semibold text-gray-900 dark:text-white',
    closeButton: 'p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500',
    content: 'p-6',
  },
  variants: {
    size: {
      sm: { container: 'max-w-sm' },
      md: { container: 'max-w-md' },
      lg: { container: 'max-w-lg' },
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
        <Dialog className="outline-none">
          {({ close }) => (
            <>
              {title && (
                <div className={styles.header()}>
                  <Heading slot="title" className={styles.title()}>
                    {title}
                  </Heading>
                  <button
                    onClick={close}
                    className={styles.closeButton()}
                    aria-label="Close"
                  >
                    <XMarkIcon className="w-5 h-5" />
                  </button>
                </div>
              )}
              <div className={styles.content()}>{children}</div>
            </>
          )}
        </Dialog>
      </AriaModal>
    </ModalOverlay>
  );
}

export { DialogTrigger };
