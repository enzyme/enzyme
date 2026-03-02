import { WebTracerProvider, BatchSpanProcessor } from '@opentelemetry/sdk-trace-web';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { getWebAutoInstrumentations } from '@opentelemetry/auto-instrumentations-web';
import { trace, SpanStatusCode } from '@opentelemetry/api';
import { getApiBase } from '@enzyme/api-client';

let initialized = false;

const MAX_ERROR_SPANS = 50;
let errorSpanCount = 0;

export function initTelemetry(): void {
  if (initialized) return;
  initialized = true;

  const endpoint = import.meta.env.VITE_OTEL_ENDPOINT ?? '/api/telemetry/traces';

  const resource = resourceFromAttributes({
    [ATTR_SERVICE_NAME]: 'enzyme-web',
    [ATTR_SERVICE_VERSION]: import.meta.env.VITE_APP_VERSION || 'dev',
  });

  const exporter = new OTLPTraceExporter({ url: endpoint });

  const provider = new WebTracerProvider({
    resource,
    spanProcessors: [new BatchSpanProcessor(exporter)],
  });

  provider.register({
    contextManager: new ZoneContextManager(),
  });

  // Build the list of allowed URLs for trace propagation.
  // Only propagate traceparent to same-origin and the configured API base.
  const apiBase = getApiBase();
  const propagateUrls: (string | RegExp)[] = [location.origin];
  if (apiBase.startsWith('http')) {
    try {
      const apiOrigin = new URL(apiBase).origin;
      if (apiOrigin !== location.origin) {
        propagateUrls.push(apiOrigin);
      }
    } catch {
      // Invalid URL — skip
    }
  }

  registerInstrumentations({
    instrumentations: [
      getWebAutoInstrumentations({
        '@opentelemetry/instrumentation-fetch': {
          propagateTraceHeaderCorsUrls: propagateUrls,
        },
        '@opentelemetry/instrumentation-xml-http-request': {
          enabled: false, // We only use fetch
        },
      }),
    ],
  });

  const tracer = trace.getTracer('enzyme-web');

  window.addEventListener('error', (event) => {
    if (++errorSpanCount > MAX_ERROR_SPANS) return;
    const span = tracer.startSpan('error.unhandled');
    span.setStatus({ code: SpanStatusCode.ERROR });
    span.recordException(event.error ?? event.message);
    if (event.filename) {
      span.setAttribute('code.filepath', event.filename);
      span.setAttribute('code.lineno', event.lineno);
      span.setAttribute('code.column', event.colno);
    }
    span.end();
  });

  window.addEventListener('unhandledrejection', (event) => {
    if (++errorSpanCount > MAX_ERROR_SPANS) return;
    const span = tracer.startSpan('error.unhandled_rejection');
    span.setStatus({ code: SpanStatusCode.ERROR });
    span.recordException(event.reason instanceof Error ? event.reason : String(event.reason));
    span.end();
  });
}
