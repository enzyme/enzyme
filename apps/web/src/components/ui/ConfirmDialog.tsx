import { type ReactNode } from 'react';
import { Dialog, Modal as AriaModal, ModalOverlay, Heading } from 'react-aria-components';
import { tv } from 'tailwind-variants';
import { Button } from './Button';

const confirmDialog = tv({
  slots: {
    overlay: [
      'fixed inset-0 z-50 flex items-center justify-center',
      'bg-black/50',
      'entering:animate-in entering:fade-in entering:duration-200',
      'exiting:animate-out exiting:fade-out exiting:duration-150',
    ],
    container: [
      'relative w-full max-w-md mx-4 bg-white dark:bg-gray-800 rounded-lg shadow-xl',
      'entering:animate-in entering:zoom-in-95 entering:duration-200',
      'exiting:animate-out exiting:zoom-out-95 exiting:duration-150',
    ],
    title: 'text-lg font-semibold text-gray-900 dark:text-white',
    description: 'mt-2 text-sm text-gray-600 dark:text-gray-300',
    actions: 'mt-6 flex justify-end gap-3',
  },
});

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: ReactNode;
  confirmLabel?: string;
  variant?: 'default' | 'destructive';
  isLoading?: boolean;
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  confirmLabel = 'Confirm',
  variant = 'default',
  isLoading = false,
}: ConfirmDialogProps) {
  const styles = confirmDialog();

  return (
    <ModalOverlay
      isOpen={isOpen}
      onOpenChange={(open) => !open && onClose()}
      isKeyboardDismissDisabled={isLoading}
      className={styles.overlay()}
    >
      <AriaModal className={styles.container()}>
        <Dialog role="alertdialog" className="p-6 outline-none">
          <Heading slot="title" className={styles.title()}>
            {title}
          </Heading>
          <div className={styles.description()}>{description}</div>
          <div className={styles.actions()}>
            <Button variant="secondary" onPress={onClose} isDisabled={isLoading}>
              Cancel
            </Button>
            <Button
              variant={variant === 'destructive' ? 'danger' : 'primary'}
              onPress={onConfirm}
              isLoading={isLoading}
              autoFocus
            >
              {confirmLabel}
            </Button>
          </div>
        </Dialog>
      </AriaModal>
    </ModalOverlay>
  );
}
