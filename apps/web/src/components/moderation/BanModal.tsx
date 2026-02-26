import { useState } from 'react';
import { Dialog, Heading, ModalOverlay, Modal as AriaModal } from 'react-aria-components';
import { ShieldExclamationIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { useNavigate } from 'react-router-dom';
import { Button, IconButton } from '../ui';
import { useAuth } from '../../hooks';
import type { WorkspaceSummary } from '@enzyme/api-client';

interface BanModalProps {
  workspace: WorkspaceSummary;
}

export function BanModal({ workspace }: BanModalProps) {
  const navigate = useNavigate();
  const { workspaces } = useAuth();
  const ban = workspace.ban!;
  const [isDismissed, setIsDismissed] = useState(false);

  const otherWorkspaces = workspaces?.filter((ws) => ws.id !== workspace.id && !ws.ban) ?? [];

  const formattedExpiry = ban.expires_at
    ? new Date(ban.expires_at).toLocaleString(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short',
      })
    : null;

  return (
    <ModalOverlay
      isOpen={!isDismissed}
      onOpenChange={(open) => {
        if (!open) setIsDismissed(true);
      }}
      isDismissable
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    >
      <AriaModal className="mx-4 w-full max-w-md rounded-lg bg-white shadow-xl dark:bg-gray-800">
        <Dialog className="relative px-6 pt-10 pb-12 outline-none">
          <div className="absolute top-3 right-3">
            <IconButton onPress={() => setIsDismissed(true)} aria-label="Close">
              <XMarkIcon className="h-4 w-4" />
            </IconButton>
          </div>
          <div className="flex flex-col items-center text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
              <ShieldExclamationIcon className="h-6 w-6 text-red-600 dark:text-red-400" />
            </div>

            <Heading
              slot="title"
              className="mt-4 text-lg font-semibold text-gray-900 dark:text-white"
            >
              You have been banned from {workspace.name}
            </Heading>

            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              {formattedExpiry ? `Expires: ${formattedExpiry}` : 'This ban is permanent'}
            </p>

            {otherWorkspaces.length > 0 && (
              <div className="mt-6 w-full">
                <Button
                  onPress={() => navigate(`/workspaces/${otherWorkspaces[0].id}`)}
                  className="w-full"
                >
                  Switch to {otherWorkspaces[0].name}
                </Button>
              </div>
            )}
          </div>
        </Dialog>
      </AriaModal>
    </ModalOverlay>
  );
}
