import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { UnstyledButton } from '../ui';
import { authApi } from '../../api';

export function EmailVerificationBanner() {
  const [dismissed, setDismissed] = useState(
    () => sessionStorage.getItem('enzyme:verification-banner-dismissed') === 'true',
  );

  const resend = useMutation({
    mutationFn: () => authApi.resendVerification(),
  });

  if (dismissed) return null;

  const handleDismiss = () => {
    sessionStorage.setItem('enzyme:verification-banner-dismissed', 'true');
    setDismissed(true);
  };

  return (
    <div className="flex flex-shrink-0 items-center justify-center gap-2 border-b border-amber-200 bg-amber-100 px-4 py-1.5 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
      <span>
        Please verify your email address.{' '}
        {resend.isPending ? (
          'Sending verification email...'
        ) : resend.isSuccess ? (
          'We re-sent the verification email.'
        ) : resend.isError ? (
          <>
            Failed to send.{' '}
            <UnstyledButton
              onPress={() => resend.mutate()}
              className="cursor-pointer font-medium underline"
            >
              Try again
            </UnstyledButton>
            .
          </>
        ) : (
          <>
            Check your inbox or{' '}
            <UnstyledButton
              onPress={() => resend.mutate()}
              className="cursor-pointer font-medium underline"
            >
              resend verification email
            </UnstyledButton>
            .
          </>
        )}
      </span>
      <UnstyledButton
        onPress={handleDismiss}
        className="ml-2 cursor-pointer rounded p-0.5 hover:bg-amber-200 dark:hover:bg-amber-800"
        aria-label="Dismiss"
      >
        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </UnstyledButton>
    </div>
  );
}
