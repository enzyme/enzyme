# Load Tests

Performance and stress tests using [K6](https://grafana.com/docs/k6/), written in TypeScript.

## Prerequisites

```bash
brew install k6   # macOS
```

The server must be running with seed data and rate limiting disabled:

```bash
make seed
make dev   # in another terminal
```

The auth test exceeds the default rate limits (10 logins/min, 5 registrations/hr per IP). Disable rate limiting in your config for load testing:

```yaml
# config.yaml
rate_limit:
  enabled: false
```

## Running

```bash
# Run all load tests (builds automatically)
make load-test

# Run individual test suites
make load-test-auth         # Authentication (login/register)
make load-test-messaging    # Messages (send/list/react/search)
make load-test-sse          # SSE connections (100 concurrent, ~80s)
make load-test-full         # Realistic mixed workflow

# SSE stress test (not included in load-test — runs longer)
make load-test-sse-stress                        # 100 connections, 2m
make load-test-sse-stress SSE_CONNECTIONS=2000   # 2000 connections

# Against a remote server
make load-test K6_BASE_URL=https://chat.enzyme.im
make load-test-sse-stress K6_BASE_URL=https://chat.enzyme.im SSE_CONNECTIONS=2000
```

## Development

```bash
# Build TypeScript to JavaScript (required before running k6)
pnpm --filter @enzyme/load-tests build

# Type-check without building
pnpm --filter @enzyme/load-tests typecheck
```

## Test Suites

| File            | What it tests                                                | Key metrics                    |
| --------------- | ------------------------------------------------------------ | ------------------------------ |
| `auth.ts`       | Login throughput, registration burst                         | login/register latency, errors |
| `messaging.ts`  | Message send/list/search under concurrent write load         | SQLite contention, p95 latency |
| `sse.ts`        | SSE connections and event delivery (~100 concurrent)         | connection errors, event count |
| `sse-stress.ts` | Thousands of SSE connections + fan-out latency measurement   | p50/p95 latency, throughput    |
| `full.ts`       | Realistic mixed workflow (browse, read, send, react, search) | end-to-end workflow latency    |

## SSE Stress Test

The stress test (`sse-stress.ts`) holds many concurrent SSE connections while sending messages, reactions, and typing indicators. It measures end-to-end fan-out latency: the time from sending a message via the API to receiving it as an SSE event.

Configuration via environment variables:

| Variable          | Default | Description                                |
| ----------------- | ------- | ------------------------------------------ |
| `SSE_CONNECTIONS` | `100`   | Number of concurrent SSE connections       |
| `SSE_DURATION`    | `2m`    | How long to hold connections after ramp-up |
| `SSE_MSG_RATE`    | `5`     | Messages per second during activity phase  |
| `SSE_RAMP`        | `30s`   | Ramp-up duration                           |

SSE support uses the [xk6-sse](https://github.com/phymbert/xk6-sse) extension, which K6 resolves automatically at runtime.

## Cleanup

The `auth` test creates `loadtest-*` user accounts and the `messaging`/`full` tests write messages into real channels on each run. These persist across runs. Re-seed between test sessions to reset:

```bash
rm server/enzyme.db && make seed
```

## Remote Targets

By default, load tests refuse to run against non-localhost servers. To run against a remote target, set `K6_CONFIRM_REMOTE=1`:

```bash
make load-test K6_BASE_URL=https://chat.enzyme.im K6_FLAGS="--env K6_CONFIRM_REMOTE=1"
```
