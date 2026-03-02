import { useEffect, useRef, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Button as AriaButton } from 'react-aria-components';
import { authApi } from '../../api';

export function EmailVerificationBanner() {
  const [dismissed, setDismissed] = useState(
    () => sessionStorage.getItem('enzyme:verification-banner-dismissed') === 'true',
  );
  const [showSuccess, setShowSuccess] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const resend = useMutation({
    mutationFn: () => authApi.resendVerification(),
    onSuccess: () => {
      setShowSuccess(true);
      timerRef.current = setTimeout(() => setShowSuccess(false), 3000);
    },
  });

  if (dismissed) return null;

  const handleDismiss = () => {
    sessionStorage.setItem('enzyme:verification-banner-dismissed', 'true');
    setDismissed(true);
  };

  return (
    <div className="flex flex-shrink-0 items-center justify-center gap-2 border-b border-amber-200 bg-amber-100 px-4 py-1.5 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
      <span>
        Please verify your email address. Check your inbox or{' '}
        <AriaButton
          onPress={() => resend.mutate()}
          isDisabled={resend.isPending || showSuccess}
          className="font-medium underline hover:no-underline disabled:opacity-50"
        >
          {resend.isPending ? 'Sending...' : showSuccess ? 'Sent!' : 'resend verification email'}
        </AriaButton>
        .
      </span>
      <AriaButton
        onPress={handleDismiss}
        className="ml-2 rounded p-0.5 hover:bg-amber-200 dark:hover:bg-amber-800"
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
      </AriaButton>
    </div>
  );
}
