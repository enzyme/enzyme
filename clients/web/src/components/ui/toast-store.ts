interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info';
}

let toastId = 0;
const listeners: Set<(toast: Toast) => void> = new Set();

export function toast(message: string, type: Toast['type'] = 'info') {
  const id = String(++toastId);
  listeners.forEach((listener) => listener({ id, message, type }));
}

export function subscribe(listener: (toast: Toast) => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export type { Toast };
