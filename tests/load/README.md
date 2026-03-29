# Load Tests

Performance and stress tests using [K6](https://grafana.com/docs/k6/).

## Prerequisites

```bash
brew install k6   # macOS
```

The server must be running with seed data:

```bash
make seed
make dev   # in another terminal
```

## Running

```bash
# Run all load tests sequentially
make load-test

# Run individual test suites
make load-test-auth         # Authentication (login/register)
make load-test-messaging    # Messages (send/list/react/search)
make load-test-sse          # SSE connections (concurrent subscribers)
make load-test-full         # Realistic mixed workflow

# Against the hosted instance
make load-test-full BASE_URL=https://chat.enzyme.im

# With extra K6 flags (e.g., summary export)
make load-test-full K6_FLAGS="--summary-export=results.json"
```

Or run K6 directly:

```bash
k6 run tests/load/auth.js
k6 run tests/load/messaging.js --env K6_BASE_URL=https://chat.enzyme.im
```

## Test Suites

| File             | What it tests                                             | Key metrics                    |
| ---------------- | --------------------------------------------------------- | ------------------------------ |
| `auth.js`        | Login throughput, registration burst                      | login/register latency, errors |
| `messaging.js`   | Message send/list/search under concurrent write load      | SQLite contention, p95 latency |
| `sse.js`         | Concurrent SSE connections, event delivery under load     | max connections, connect time  |
| `full.js`        | Realistic mixed workflow (browse, read, send, react, search) | end-to-end workflow latency |

## Thresholds

Each test defines pass/fail thresholds. K6 exits non-zero if thresholds are breached:

- **Error rate**: < 1% HTTP failures
- **p95 latency**: < 500ms for most endpoints, < 800ms for writes, < 1s for search
- **SSE connections**: < 10 connection failures

## Architecture Notes

These tests specifically stress:

- **SQLite single-connection bottleneck**: `messaging.js` runs 20-30 concurrent writers to expose write contention from `SetMaxOpenConns(1)`
- **SSE hub memory**: `sse.js` ramps to 100 concurrent SSE connections while generating events
- **Realistic read/write ratio**: `full.js` mixes ~70% reads with ~30% writes, matching typical chat app usage

## Files

- `helpers.js` — Shared config, auth helpers, API wrappers, threshold presets
- `auth.js` — Authentication load test
- `messaging.js` — Messaging load test
- `sse.js` — SSE stress test
- `full.js` — Full workflow load test
