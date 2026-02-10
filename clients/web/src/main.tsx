import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { setApiBase } from '@feather/api-client';
import './index.css';
import App from './App.tsx';

const apiBase = import.meta.env.VITE_API_BASE;
if (apiBase) {
  setApiBase(apiBase);
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
