import { setTokenStorage } from '@enzyme/api-client';

const TOKEN_KEY = 'enzyme_auth_token';

// Plug in localStorage-backed persistence at module load
setTokenStorage({
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (token) => localStorage.setItem(TOKEN_KEY, token),
  remove: () => localStorage.removeItem(TOKEN_KEY),
});

export { useAuth } from '@enzyme/shared';
