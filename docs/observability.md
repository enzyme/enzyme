# Observability Guide

Enzyme has built-in OpenTelemetry (OTel) instrumentation for traces and metrics. When enabled, telemetry data is exported via OTLP to any compatible backend — Jaeger, Grafana, Datadog, Honeycomb, etc. Disabled by default with zero overhead.

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

| Key                      | Env Var                         | Default          | Description                                                |
| ------------------------ | ------------------------------- | ---------------- | ---------------------------------------------------------- |
| `telemetry.enabled`      | `ENZYME_TELEMETRY_ENABLED`      | `false`          | Enable OpenTelemetry instrumentation.                      |
| `telemetry.endpoint`     | `ENZYME_TELEMETRY_ENDPOINT`     | `localhost:4317` | OTLP collector endpoint (host:port).                       |
| `telemetry.protocol`     | `ENZYME_TELEMETRY_PROTOCOL`     | `grpc`           | Export protocol: `grpc` (port 4317) or `http` (port 4318). |
| `telemetry.sample_rate`  | `ENZYME_TELEMETRY_SAMPLE_RATE`  | `1.0`            | Trace sampling rate. `1.0` = all, `0.1` = 10%, `0` = none. |
| `telemetry.service_name` | `ENZYME_TELEMETRY_SERVICE_NAME` | `enzyme`         | Service name reported to the collector.                    |

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

| Metric                   | Type          | Attributes            | Description                              |
| ------------------------ | ------------- | --------------------- | ---------------------------------------- |
| `sse.connections.active` | UpDownCounter | —                     | Current number of active SSE connections |
| `sse.events.broadcast`   | Counter       | `event.type`, `scope` | Total SSE events broadcast               |

**`sse.events.broadcast` attributes:**

- `event.type`: The SSE event type (e.g., `message.new`, `typing.start`, `presence.changed`)
- `scope`: Either `workspace` (broadcast to all members) or `channel` (broadcast to channel members only)

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

## Setup Examples

### Jaeger (Local Development)

Jaeger all-in-one accepts OTLP on port 4317 (gRPC) and 4318 (HTTP), and provides a UI on port 16686.

```bash
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

```bash
ENZYME_TELEMETRY_ENABLED=true ./enzyme
```

Open `http://localhost:16686` and select the `enzyme` service to view traces.

### Grafana + Tempo + OTel Collector

For a production-grade setup with Grafana dashboards, Tempo for traces, and Prometheus for metrics:

```yaml
# docker-compose.yml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    volumes:
      - ./otel-config.yaml:/etc/otelcol/config.yaml
    ports:
      - '4317:4317' # OTLP gRPC
      - '4318:4318' # OTLP HTTP

  tempo:
    image: grafana/tempo:latest
    command: ['-config.file=/etc/tempo.yaml']
    volumes:
      - ./tempo.yaml:/etc/tempo.yaml

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - '3001:3000'
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
```

```yaml
# otel-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

  # System metrics from the host
  hostmetrics:
    collection_interval: 30s
    scrapers:
      cpu:
      memory:
      disk:
      network:

exporters:
  otlphttp/tempo:
    endpoint: http://tempo:4318

  prometheusremotewrite:
    endpoint: http://prometheus:9090/api/v1/write

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlphttp/tempo]
    metrics:
      receivers: [otlp, hostmetrics]
      exporters: [prometheusremotewrite]
```

### Datadog

Datadog Agent accepts OTLP natively:

```bash
# In your Datadog Agent config (datadog.yaml)
otlp_config:
  receiver:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
```

```bash
ENZYME_TELEMETRY_ENABLED=true \
ENZYME_TELEMETRY_ENDPOINT=localhost:4317 \
ENZYME_TELEMETRY_SERVICE_NAME=enzyme-production \
./enzyme
```

---

## System Metrics with the OTel Collector

Enzyme's built-in telemetry covers application-level signals (HTTP requests, database queries, SSE connections). For **system-level metrics** like CPU usage, memory, disk I/O, and network throughput, run an OpenTelemetry Collector alongside Enzyme with the `hostmetrics` receiver.

### Why a Separate Collector?

System metrics (CPU, memory, disk) are properties of the host, not the application. The OTel Collector's `hostmetrics` receiver is purpose-built for this and runs as a lightweight sidecar. This also gives you a single point to route, filter, and transform all telemetry data before it reaches your backend.

### Minimal Collector Config

```yaml
# /etc/otelcol/config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

  hostmetrics:
    collection_interval: 30s
    scrapers:
      cpu:
        metrics:
          system.cpu.utilization:
            enabled: true
      memory:
        metrics:
          system.memory.utilization:
            enabled: true
      disk:
      filesystem:
      network:
      load:

processors:
  batch:
    timeout: 10s

exporters:
  # Replace with your backend
  otlphttp:
    endpoint: https://your-backend:4318

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp]
    metrics:
      receivers: [otlp, hostmetrics]
      processors: [batch]
      exporters: [otlphttp]
```

### Running the Collector

**Docker:**

```bash
docker run -d --name otel-collector \
  --network host \
  -v /etc/otelcol/config.yaml:/etc/otelcol/config.yaml \
  otel/opentelemetry-collector-contrib:latest
```

**systemd:**

```bash
# Download
curl -LO https://github.com/open-telemetry/opentelemetry-collector-releases/releases/latest/download/otelcol-contrib_linux_amd64.deb
dpkg -i otelcol-contrib_linux_amd64.deb

# Config is at /etc/otelcol-contrib/config.yaml
systemctl enable --now otelcol-contrib
```

### System Metrics Reference

With the `hostmetrics` receiver, you get:

| Metric                          | Description                                |
| ------------------------------- | ------------------------------------------ |
| `system.cpu.time`               | CPU time per state (user, system, idle)    |
| `system.cpu.utilization`        | CPU utilization as a ratio (0.0-1.0)       |
| `system.memory.usage`           | Memory usage by state (used, free, cached) |
| `system.memory.utilization`     | Memory utilization as a ratio              |
| `system.disk.io`                | Disk read/write bytes                      |
| `system.disk.operations`        | Disk read/write operations count           |
| `system.filesystem.usage`       | Filesystem usage (used, free)              |
| `system.filesystem.utilization` | Filesystem utilization as a ratio          |
| `system.network.io`             | Network bytes sent/received                |
| `system.network.connections`    | Network connection count by state          |
| `system.cpu.load_average.1m`    | 1-minute load average                      |
| `system.cpu.load_average.5m`    | 5-minute load average                      |
| `system.cpu.load_average.15m`   | 15-minute load average                     |

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
