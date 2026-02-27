import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { setApiBase } from '@enzyme/api-client';
import './index.css';
import App from './App.tsx';

const apiBase = import.meta.env.VITE_API_BASE;
if (apiBase) {
  setApiBase(apiBase);
}

// Initialize OpenTelemetry if enabled (lazy-loaded to avoid bundle cost when disabled)
if (import.meta.env.VITE_OTEL_ENABLED === 'true') {
  import('./lib/telemetry').then(({ initTelemetry }) => initTelemetry());
}

// Suppress the browser's native context menu app-wide
document.addEventListener('contextmenu', (e) => e.preventDefault());

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
