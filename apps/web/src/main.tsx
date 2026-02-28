import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { setApiBase } from '@enzyme/api-client';
import './index.css';
import App from './App.tsx';

const apiBase = import.meta.env.VITE_API_BASE;
if (apiBase) {
  setApiBase(apiBase);
}

// Suppress the browser's native context menu app-wide
document.addEventListener('contextmenu', (e) => e.preventDefault());

async function bootstrap() {
  // Initialize OpenTelemetry before rendering so the fetch instrumentation
  // is in place for the first API calls. Runtime config (injected by the Go
  // server) takes precedence; VITE_OTEL_ENABLED is a dev-only fallback.
  const enzymeConfig = window.__ENZYME_CONFIG__;
  if (enzymeConfig?.telemetry?.enabled || import.meta.env.VITE_OTEL_ENABLED === 'true') {
    try {
      const { initTelemetry } = await import('./lib/telemetry');
      initTelemetry({ endpoint: enzymeConfig?.telemetry?.endpoint });
    } catch (e) {
      console.warn('Failed to initialize telemetry:', e);
    }
  }

  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  );
}

bootstrap();
