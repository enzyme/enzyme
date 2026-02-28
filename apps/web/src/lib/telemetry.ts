import { WebTracerProvider, BatchSpanProcessor } from '@opentelemetry/sdk-trace-web';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { getWebAutoInstrumentations } from '@opentelemetry/auto-instrumentations-web';
import { getApiBase } from '@enzyme/api-client';

export function initTelemetry(config?: { endpoint?: string }): void {
  const endpoint = config?.endpoint ?? import.meta.env.VITE_OTEL_ENDPOINT ?? '/v1/traces';

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
}
