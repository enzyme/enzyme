# SSE Stress Test

Holds thousands of concurrent SSE connections while generating realistic traffic (messages, reactions, typing) and measuring event fan-out latency to all subscribers.

Uses the seed users (alice, bob, carol, etc.) — run `make seed` first.

## Usage

```bash
cd tests/load/sse-stress

# Default: 10,000 connections against localhost
go run . -connections 2000

# Against production
go run . -base-url https://chat.enzyme.im -connections 2000 -ramp-rate 200 -duration 2m -msg-rate 5

# Force HTTP/1.1 (for A/B testing HTTP/2 vs HTTP/1.1)
go run . -base-url https://chat.enzyme.im -connections 2000 -h1
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `-base-url` | `http://localhost:8080` | Server base URL |
| `-connections` | `10000` | Number of concurrent SSE connections |
| `-ramp-rate` | `500` | Connections to open per second during ramp-up |
| `-duration` | `2m` | How long to hold connections after ramp-up |
| `-msg-rate` | `5` | Messages per second per user during activity phase |
| `-password` | `password` | Password for seed users |
| `-h1` | `false` | Force HTTP/1.1 on SSE connections |

## Output

Prints a live stats line during the test. At the end, prints a summary with message throughput, event counts, and latency percentiles (p50, p95, p99).
