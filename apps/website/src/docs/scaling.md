---
title: 'Scaling'
description: 'Performance tuning and scaling considerations'
section: 'Self-Hosting & Operations'
order: 44
---

# Scaling Guide

Enzyme is designed for single-server deployment with SQLite. The default configuration works well on a 2 GB / 1 vCPU box (~500 concurrent users). This guide explains how to scale up.

For a full list of configurable options, see [Configuration Reference](/docs/configuration/).

---

## SQLite Tuning

SQLite handles all storage in Enzyme. These pragmas are set per-connection via DSN parameters, so every connection in the pool gets them.

| Setting              | Config Key                      | Default       | What It Does                                                                                           |
| -------------------- | ------------------------------- | ------------- | ------------------------------------------------------------------------------------------------------ |
| `max_open_conns`     | `database.max_open_conns`       | `10`          | Number of connections in the pool. With WAL mode, readers don't block writers, so >1 is safe.          |
| `busy_timeout`       | `database.busy_timeout`         | `5000`        | Milliseconds to retry when the database is locked, before returning `SQLITE_BUSY`.                     |
| `cache_size`         | `database.cache_size`           | `-8000`       | Page cache size **per connection**. Negative = KB (`-8000` = ~8 MB). Larger cache = fewer disk reads.  |
| `mmap_size`          | `database.mmap_size`            | `268435456`   | Memory-mapped I/O in bytes. `0` = disabled. Default is 256 MB. Enables the OS to page database data directly into memory. |
| `journal_size_limit` | `database.journal_size_limit`   | `67108864`    | Max WAL file size in bytes. Default is 64 MB. Caps WAL growth during heavy writes.                     |

### When to Adjust

- **More concurrent users**: Increase `max_open_conns` beyond the default of 10 if you consistently see connection pool exhaustion under high load.
- **Write contention errors**: Increase `busy_timeout`. If you see `SQLITE_BUSY` in logs, the default 5 seconds isn't enough for your write volume.
- **Slow queries on large databases**: Increase `cache_size` (e.g., `-64000` for ~64 MB) and `mmap_size` (e.g., `1073741824` for 1 GB). This keeps hot pages in memory.
- **Small VPS with limited RAM**: Lower `cache_size` (e.g., `-2000` for ~2 MB) and `mmap_size` (e.g., `0` to disable). The defaults are tuned for moderate workloads.

> **Note:** `cache_size` is per-connection. Total cache memory is roughly `cache_size × max_open_conns`. With the defaults (`-8000` and `10`), that's ~80 MB total.

---

## HTTP Server Tuning

| Setting         | Config Key             | Default | What It Does                                                                 |
| --------------- | ---------------------- | ------- | ---------------------------------------------------------------------------- |
| `read_timeout`  | `server.read_timeout`  | `30s`   | Max time to read the full request (headers + body).                          |
| `write_timeout` | `server.write_timeout` | `60s`   | Max time to write the response. SSE connections disable this per-connection. |
| `idle_timeout`  | `server.idle_timeout`  | `120s`  | How long to keep idle keep-alive connections open.                           |

### When to Adjust

- **Large file uploads over slow connections**: Increase `read_timeout` (e.g., `120s`).
- **Clients on high-latency networks**: Increase `write_timeout`.
- **Many idle connections consuming file descriptors**: Decrease `idle_timeout`.

---

## SSE Tuning

| Setting              | Config Key               | Default | What It Does                                                          |
| -------------------- | ------------------------ | ------- | --------------------------------------------------------------------- |
| `heartbeat_interval` | `sse.heartbeat_interval` | `30s`   | How often heartbeat events are sent to keep connections alive.        |
| `client_buffer_size` | `sse.client_buffer_size` | `256`   | Go channel buffer per connected SSE client.                           |
| `event_retention`    | `sse.event_retention`    | `24h`   | How long events are stored in the database for reconnection catch-up. |

### When to Adjust

- **High-traffic workspaces** (many messages/second): Increase `client_buffer_size` (e.g., `512` or `1024`). If the buffer fills, the slow client misses events and must reconnect.
- **Aggressive proxies/load balancers dropping idle connections**: Decrease `heartbeat_interval` (e.g., `15s`).
- **Database growing too large from event storage**: Decrease `event_retention`.

---

## OS-Level Tuning

### File Descriptors

Each SSE connection uses a file descriptor. The default Linux limit (1024) can be a bottleneck with many concurrent users. Add to your systemd unit:

```ini
[Service]
LimitNOFILE=65536
```

Or set system-wide in `/etc/security/limits.conf`:

```
enzyme  soft  nofile  65536
enzyme  hard  nofile  65536
```

### TCP Memory

Under heavy broadcast load, the kernel's TCP buffer memory can hit the pressure threshold and start dropping connections. This manifests as periodic waves of disconnections followed by reconnections.

Check current TCP memory usage:

```bash
cat /proc/net/sockstat | grep TCP
# TCP: inuse 5012 ... mem 18432
```

The `mem` value is in pages (4 KB each). Compare it to the kernel's pressure threshold:

```bash
sysctl net.ipv4.tcp_mem
# net.ipv4.tcp_mem = 383139  510854  766278
#                    min      pressure  max (in pages)
```

If `mem` approaches the pressure value during load, increase the limits:

```bash
sysctl -w net.ipv4.tcp_mem="50000 75000 100000"
```

Add to `/etc/sysctl.conf` to persist across reboots. On a 1 GB server with 5,000 SSE connections, TCP memory can peak at ~80 MB (20,000 pages) during broadcast bursts.

### TCP Backlog

For servers handling many simultaneous connections:

```bash
sysctl -w net.core.somaxconn=4096
sysctl -w net.ipv4.tcp_max_syn_backlog=4096
sysctl -w net.core.netdev_max_backlog=5000
```

The default `tcp_max_syn_backlog` of 128 is too low — if connections drop and thousands of clients reconnect simultaneously, the SYN queue overflows and reconnections fail.

Add to `/etc/sysctl.conf` to persist across reboots.

### Capabilities and SQLite

If you use `setcap cap_net_bind_service=+ep` on the Enzyme binary to bind to port 443 without root, be aware that file capabilities interact badly with `modernc.org/sqlite` (the pure-Go SQLite driver). The capability flag changes how the kernel handles the process's memory mappings, causing disk I/O errors on all write operations.

Instead, grant the capability through systemd:

```ini
[Service]
User=enzyme
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
```

This achieves the same result without modifying the binary.

### Profiling

To investigate performance issues, you can add Go's built-in pprof profiler. Import `net/http/pprof` and start a listener on a localhost-only port in a goroutine:

```go
import "net/http/pprof"

mux := http.NewServeMux()
mux.HandleFunc("/debug/pprof/", pprof.Index)
mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
go http.ListenAndServe("localhost:6060", mux)
```

Then access it via SSH tunnel:

```bash
ssh -L 6060:localhost:6060 root@your-server
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30  # CPU
go tool pprof http://localhost:6060/debug/pprof/heap                # memory
```

---

## SSE Performance

Enzyme broadcasts real-time events to all connected clients via Server-Sent Events. At high connection counts, three bottlenecks can appear:

### Per-subscriber serialization

Each SSE event is serialized to JSON once before broadcast and sent as pre-marshaled bytes to all subscribers. If you see high CPU in `json.Marshal` under load, this is already optimized in current versions.

### HTTP/2 vs HTTP/1.1

HTTP/2 multiplexes streams over a shared TCP connection, while HTTP/1.1 gives each SSE client its own socket. In practice, real browsers open one SSE connection per user (not multiple streams over one TCP connection), so HTTP/2's multiplexing doesn't affect SSE fan-out. Enzyme leaves HTTP/2 enabled (Go's default) — testing showed no performance difference for SSE workloads.

### Flush overhead

SSE writes are batched — when an event arrives, all pending events in the client's buffer are drained and written before a single `Flush()` call. This reduces syscall overhead under burst traffic. The `client_buffer_size` setting controls how many events can queue before the client is considered too slow.

---

## Reverse Proxy Tuning

See the [Self-Hosting Guide](/docs/self-hosting/#advanced-reverse-proxy) for nginx and Caddy configuration examples. The key considerations for scaling:

- **Disable response buffering** for SSE endpoints (`proxy_buffering off` in nginx, `flush_interval -1` in Caddy)
- **Set long read timeouts** on SSE paths (e.g., `proxy_read_timeout 86400s`) to prevent the proxy from killing idle SSE connections
- **Increase worker connections** in nginx (`worker_connections 4096`) if you expect thousands of concurrent SSE clients

---

## Example Configs

### Small (2 GB / 1 vCPU, ~500 users)

Use the defaults. No config changes needed. A single vCPU comfortably handles ~2,000 SSE connections, and real-world message rates are far lower than stress-test conditions.

### Medium (8 GB / 4 vCPU, ~5,000 users)

```yaml
database:
  max_open_conns: 4
  busy_timeout: 10000
  cache_size: -16000 # ~16 MB per conn, ~64 MB total
  mmap_size: 134217728 # 128 MB

server:
  read_timeout: '60s'
  write_timeout: '120s'
  idle_timeout: '120s'

sse:
  client_buffer_size: 512
```

### Large (16 GB / 8 vCPU, ~20,000 users)

```yaml
database:
  max_open_conns: 8
  busy_timeout: 15000
  cache_size: -64000 # ~64 MB per conn, ~512 MB total
  mmap_size: 1073741824 # 1 GB

server:
  read_timeout: '60s'
  write_timeout: '120s'
  idle_timeout: '300s'

sse:
  heartbeat_interval: '20s'
  client_buffer_size: 1024

telemetry:
  sample_rate: 0.1 # 10% — full sampling is too expensive at this scale
```

Also set `LimitNOFILE=65536` in the systemd unit for this profile.

---

## Telemetry Sampling

If [OpenTelemetry is enabled](/docs/observability/), the default `sample_rate` of `1.0` traces every request. This is fine for small deployments but adds overhead at scale — each trace generates spans for the HTTP request, database queries, and SSE operations.

For the Medium profile (~1,000 users), `0.5` (50%) is a reasonable starting point. For the Large profile (~10,000 users), drop to `0.1` (10%) or lower. You can also disable tracing entirely (`0.0`) and keep only metrics, which are always aggregated and cheap regardless of traffic volume.

```yaml
telemetry:
  enabled: true
  sample_rate: 0.1 # adjust based on traffic
```

---

## Monitoring

Key metrics to watch when scaling:

- **SQLite busy retries**: `SQLITE_BUSY` errors in logs indicate write contention. Increase `busy_timeout` or reduce `max_open_conns`. Occasional SQLITE_BUSY under peak load is normal — the important thing is that they don't cascade into persistent I/O errors (which would indicate an outdated `modernc.org/sqlite` version; v1.46.1+ is required).
- **SSE connection count**: Monitor the number of active SSE clients. Each consumes memory proportional to `client_buffer_size`.
- **Memory usage**: (`cache_size` x `max_open_conns`) + `mmap_size` + (SSE clients x buffer size x avg event size) gives a rough memory floor.
- **File descriptors**: `ls /proc/$(pidof enzyme)/fd | wc -l` shows current usage. Compare to `LimitNOFILE`.
- **Response latency**: If P99 response times degrade, check whether `max_open_conns` is sufficient. Expensive read queries (like notification aggregation) can hold connections for hundreds of milliseconds, starving other handlers. If latency persists, the database may benefit from a larger cache or mmap. Use `EXPLAIN QUERY PLAN` on slow queries to verify they're using indexes.
- **TCP memory**: During SSE broadcast bursts, check `cat /proc/net/sockstat | grep TCP` — if the `mem` value approaches the kernel's `tcp_mem` pressure threshold, connections will be dropped. See [TCP Memory](#tcp-memory) above.
- **Trace export backpressure**: If the OTLP exporter queue fills up, you'll see dropped spans in logs. Lower `sample_rate` or scale your collector.
