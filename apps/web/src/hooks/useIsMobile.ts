import { useSyncExternalStore } from 'react';

// Must match Tailwind's `md` breakpoint (768px). max-width: 767px = "below md".
const mql = typeof window !== 'undefined' ? window.matchMedia('(max-width: 767px)') : null;

const subscribe = (cb: () => void) => {
  mql?.addEventListener('change', cb);
  return () => mql?.removeEventListener('change', cb);
};
const getSnapshot = () => mql?.matches ?? false;
const getServerSnapshot = () => false;

export function useIsMobile() {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
