# Observability Guide

Enzyme has built-in OpenTelemetry (OTel) instrumentation for traces and metrics. When enabled, telemetry data is exported via OTLP to any compatible backend — Datadog, Honeycomb, HyperDX, Grafana Cloud, etc. Disabled by default with zero overhead.

This guide covers what Enzyme captures, how to set it up, and how to add system-level metrics with an OTel Collector.

## Quick Start

Enable telemetry with a single environment variable:

```bash
ENZYME_TELEMETRY_ENABLED=true ./enzyme
```

Or in `config.yaml`:

```yaml
telemetry:
  enabled: true
  endpoint: 'localhost:4317'
  protocol: 'grpc'
  sample_rate: 1.0
  service_name: 'enzyme'
```

Enzyme pushes data to the configured OTLP endpoint. You need a collector or backend listening there to receive it. See [Setup Examples](#setup-examples) below.

## Configuration

| Key                      | Env Var                         | Default          | Description                                                          |
| ------------------------ | ------------------------------- | ---------------- | -------------------------------------------------------------------- |
| `telemetry.enabled`      | `ENZYME_TELEMETRY_ENABLED`      | `false`          | Enable OpenTelemetry instrumentation.                                |
| `telemetry.endpoint`     | `ENZYME_TELEMETRY_ENDPOINT`     | `localhost:4317` | OTLP collector endpoint (host:port).                                 |
| `telemetry.protocol`     | `ENZYME_TELEMETRY_PROTOCOL`     | `grpc`           | Export protocol: `grpc` (port 4317) or `http` (port 4318).           |
| `telemetry.insecure`     | `ENZYME_TELEMETRY_INSECURE`     | `true`           | Use plaintext (no TLS) for OTLP export.                              |
| `telemetry.sample_rate`  | `ENZYME_TELEMETRY_SAMPLE_RATE`  | `1.0`            | Trace sampling rate. `1.0` = all, `0.1` = 10%, `0` = none.           |
| `telemetry.service_name` | `ENZYME_TELEMETRY_SERVICE_NAME` | `enzyme`         | Service name reported to the collector.                              |
| `telemetry.headers`      |                                 |                  | Map of headers sent with every OTLP export request (e.g., API keys). |

See the full [Configuration Reference](configuration.md#telemetry-opentelemetry) for details.

---

## What's Captured

### Traces

Traces show the lifecycle of a request from start to finish. Each trace is a tree of **spans** representing operations.

#### HTTP Request Spans

Every incoming HTTP request creates a root span with:

- **Span name**: `METHOD /route/pattern` (e.g., `GET /api/workspaces/{wid}/channels`)
- **Attributes**: HTTP method, status code, route pattern, response size, user agent
- **Duration**: Total request processing time

The span name uses the route pattern (not the raw URL), so `/api/workspaces/01J5X.../channels` appears as `/api/workspaces/{wid}/channels`. This keeps span cardinality manageable.

#### Database Spans

Key database operations create child spans under the HTTP request span. These help identify slow queries:

| Span Name                  | Operation                                       |
| -------------------------- | ----------------------------------------------- |
| `message.Create`           | Insert a new message (transaction)              |
| `message.List`             | List messages in a channel (paginated)          |
| `message.Search`           | Full-text search across workspace messages      |
| `channel.GetByID`          | Fetch a single channel                          |
| `channel.ListForWorkspace` | List all channels with membership info (JOIN)   |
| `workspace.GetMembership`  | Check user's membership and role in a workspace |

All database spans include the `db.system: sqlite` attribute.

#### Trace Context Propagation

When frontend telemetry is enabled, the browser injects a W3C `traceparent` header into API requests. The backend extracts this header and creates child spans under the frontend's trace, giving you end-to-end visibility from button click to database query.

The CORS configuration automatically allows `traceparent` and `tracestate` headers when telemetry is enabled.

### Metrics

Metrics are exported every 60 seconds via OTLP.

| Metric                   | Type          | Attributes | Description                              |
| ------------------------ | ------------- | ---------- | ---------------------------------------- |
| `sse.connections.active` | UpDownCounter | —          | Current number of active SSE connections |
| `sse.events.broadcast`   | Counter       | `scope`    | Total SSE events broadcast               |

**`sse.events.broadcast` attributes:**

- `scope`: `workspace` (broadcast to all members), `channel` (broadcast to channel members only), or `user` (targeted to a single user)

### Log Correlation

When telemetry is enabled, every log line is enriched with `trace_id` and `span_id` fields from the active request context. This lets you jump from a log entry directly to the corresponding trace in your observability backend.

Example log output with telemetry enabled (JSON format):

```json
{
  "time": "2026-02-27T10:15:30Z",
  "level": "INFO",
  "msg": "request completed",
  "method": "POST",
  "path": "/api/workspaces/01J5X/channels/01J6Y/messages",
  "status": 200,
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

Text format also includes the fields:

```
time=2026-02-27T10:15:30Z level=INFO msg="request completed" method=POST path=... trace_id=4bf92f3577b34da6a3ce929d0e0e4736 span_id=00f067aa0ba902b7
```

### Resource Attributes

Every trace and metric is tagged with resource attributes that identify the Enzyme instance:

| Attribute         | Value                                         |
| ----------------- | --------------------------------------------- |
| `service.name`    | Configured `service_name` (default: `enzyme`) |
| `service.version` | Server version (from build, e.g., `0.1.0`)    |
| `host.name`       | Hostname of the machine                       |
| `os.type`         | Operating system (e.g., `linux`, `darwin`)    |

---

## Frontend Telemetry

The web client has separate, opt-in telemetry. When enabled, it instruments browser-side fetch calls and page loads, and propagates trace context to the backend.

### Enabling

Set these environment variables **at build time** (they're baked into the Vite bundle):

```bash
VITE_OTEL_ENABLED=true VITE_OTEL_ENDPOINT=/v1/traces pnpm --filter @enzyme/web build
```

| Env Var              | Default      | Description                                   |
| -------------------- | ------------ | --------------------------------------------- |
| `VITE_OTEL_ENABLED`  | not set      | Set to `"true"` to enable frontend telemetry. |
| `VITE_OTEL_ENDPOINT` | `/v1/traces` | OTLP HTTP endpoint for trace export.          |

### What's Instrumented

- **Fetch calls**: Every `fetch()` request creates a span with URL, method, status, and duration. Since the `@enzyme/api-client` uses native `fetch`, all API calls are automatically covered.
- **Page loads**: Document load timing (DNS, TCP, TLS, TTFB, DOM content loaded, load complete).
- **Trace propagation**: W3C `traceparent` header is injected into requests to same-origin and the configured API base URL. Third-party requests are not affected.

### How It Works

Frontend telemetry is **lazy-loaded** — the OTel SDK is only imported when `VITE_OTEL_ENABLED` is `"true"`. When disabled, zero OTel code is included in the bundle.

The frontend reports as service name `enzyme-web`, separate from the backend's `enzyme` service. In your observability backend, you'll see traces that start in `enzyme-web` and continue into `enzyme` via the propagated trace context.

### Collector Routing

The frontend exports traces over OTLP/HTTP (not gRPC, since browsers can't speak gRPC). The default endpoint `/v1/traces` assumes you either:

1. Proxy `/v1/traces` to your OTel Collector via your reverse proxy (nginx, Caddy, etc.)
2. Set `VITE_OTEL_ENDPOINT` to the full collector URL (e.g., `https://otel.example.com:4318/v1/traces`)

---

## Sampling

For high-traffic deployments, trace sampling reduces data volume without losing visibility. The `sample_rate` setting controls what fraction of traces are captured:

| Value | Effect                                                             |
| ----- | ------------------------------------------------------------------ |
| `1.0` | Sample everything (default). Good for development and low-traffic. |
| `0.5` | Sample 50% of traces. Good balance for moderate traffic.           |
| `0.1` | Sample 10%. Suitable for high-traffic production.                  |
| `0.0` | Disable tracing entirely. Metrics are still exported.              |

Sampling is **parent-based**: if an incoming request already has a `traceparent` header with a sampling decision, Enzyme respects it. This ensures that frontend-initiated traces are complete even at low sample rates.

## Graceful Shutdown

When Enzyme receives SIGINT or SIGTERM, it flushes all pending traces and metrics to the collector before shutting down. The flush happens within the 30-second shutdown timeout. No data is lost during normal restarts or deployments.

## Performance Impact

When telemetry is **disabled** (the default), there is zero overhead — no providers are initialized, no metrics are recorded, and no spans are created.

When **enabled**, the overhead is minimal:

- HTTP middleware adds ~1-2 microseconds per request for span creation
- Database spans add ~0.5 microseconds per instrumented operation
- SSE metrics use atomic counter operations (negligible cost)
- Traces are batched and exported asynchronously in the background
- Metrics are collected and exported every 60 seconds

For most deployments, telemetry overhead is unmeasurable relative to actual request processing time.

---

## Setup Examples

Enzyme exports standard OTLP, so it works with any backend that accepts OTLP. Below are examples for common platforms.

The Honeycomb and HyperDX examples send OTLP directly from Enzyme to the backend. This covers Enzyme's own traces and metrics, but not system-level metrics like CPU, memory, and disk usage. If you want those, run an [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) alongside Enzyme with the [Host Metrics receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver) enabled, point Enzyme at the collector, and have the collector forward to your backend. The Datadog example already works this way since the Datadog Agent acts as a collector.

### Datadog

The Datadog Agent runs alongside Enzyme and accepts OTLP locally, then forwards to Datadog. No API key header is needed from Enzyme — the agent holds the key.

Enable the OTLP receiver in your Datadog Agent config:

```yaml
# datadog.yaml
otlp_config:
  receiver:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
```

Then point Enzyme at the agent:

```yaml
telemetry:
  enabled: true
  endpoint: 'localhost:4317'
```

If the agent runs on a different host (e.g., a sidecar in Kubernetes), replace `localhost` with the agent's address.

### Honeycomb

Honeycomb accepts OTLP directly — no collector or agent needed. Authentication uses the `x-honeycomb-team` header with your API key.

```yaml
telemetry:
  enabled: true
  endpoint: 'api.honeycomb.io:443'
  protocol: 'grpc'
  insecure: false
  headers:
    x-honeycomb-team: 'YOUR_API_KEY'
```

For EU region, use `api.eu1.honeycomb.io:443` instead.

### HyperDX

HyperDX accepts OTLP directly. Authentication uses the `authorization` header with your ingestion API key (found under Team Settings in HyperDX).

```yaml
telemetry:
  enabled: true
  endpoint: 'in-otel.hyperdx.io:4317'
  protocol: 'grpc'
  insecure: false
  headers:
    authorization: 'YOUR_HYPERDX_API_KEY'
```

### Other Backends

Any OTLP-compatible backend (Axiom, Grafana Cloud, New Relic, etc.) works the same way: set `telemetry.endpoint` to their OTLP receiver, `telemetry.protocol` to `grpc` or `http`, and pass any required auth headers via `telemetry.headers`.
