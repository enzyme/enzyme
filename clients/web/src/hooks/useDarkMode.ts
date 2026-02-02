import { useCallback, useEffect } from 'react';
import { useLocalStorage } from './useLocalStorage';

const DARK_MODE_KEY = 'feather:dark-mode';

export type ThemeMode = 'system' | 'light' | 'dark';

/**
 * Get system color scheme preference
 */
function getSystemPreference(): boolean {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

/**
 * localStorage-backed dark mode with system preference fallback
 */
export function useDarkMode() {
  // null means "follow system"
  const [preference, setPreference] = useLocalStorage<boolean | null>(DARK_MODE_KEY, null);

  // Derive mode from stored preference
  const mode: ThemeMode = preference === null ? 'system' : preference ? 'dark' : 'light';

  // Resolve actual dark mode value
  const darkMode = preference ?? getSystemPreference();

  // Update DOM class
  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode);
  }, [darkMode]);

  // Listen for system preference changes (only when following system)
  useEffect(() => {
    if (preference !== null) return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => {
      document.documentElement.classList.toggle('dark', mediaQuery.matches);
    };
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [preference]);

  const toggle = useCallback(() => {
    setPreference(prev => {
      const current = prev ?? getSystemPreference();
      return !current;
    });
  }, [setPreference]);

  const setDarkMode = useCallback((dark: boolean) => {
    setPreference(dark);
  }, [setPreference]);

  const setMode = useCallback((newMode: ThemeMode) => {
    switch (newMode) {
      case 'system':
        setPreference(null);
        break;
      case 'light':
        setPreference(false);
        break;
      case 'dark':
        setPreference(true);
        break;
    }
  }, [setPreference]);

  return { darkMode, mode, toggle, setDarkMode, setMode };
}
