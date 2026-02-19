# Scaling Guide

Enzyme is designed for single-server deployment with SQLite. The default configuration is tuned for a 2 GB / 1 vCPU box (~100 concurrent users). This guide explains how to scale up.

For a full list of configurable options, see [Configuration Reference](configuration.md).

---

## SQLite Tuning

SQLite handles all storage in Enzyme. These pragmas are set per-connection via DSN parameters, so every connection in the pool gets them.

| Setting          | Config Key                | Default | What It Does                                                                                           |
| ---------------- | ------------------------- | ------- | ------------------------------------------------------------------------------------------------------ |
| `max_open_conns` | `database.max_open_conns` | `2`     | Number of connections in the pool. With WAL mode, readers don't block writers, so >1 is safe.          |
| `busy_timeout`   | `database.busy_timeout`   | `5000`  | Milliseconds to retry when the database is locked, before returning `SQLITE_BUSY`.                     |
| `cache_size`     | `database.cache_size`     | `-2000` | Page cache size **per connection**. Negative = KB (`-2000` = ~2 MB). Larger cache = fewer disk reads.  |
| `mmap_size`      | `database.mmap_size`      | `0`     | Memory-mapped I/O in bytes. `0` = disabled. Enables the OS to page database data directly into memory. |

### When to Adjust

- **More concurrent users**: Increase `max_open_conns` (4-8 is reasonable for most workloads). Each read query can run on its own connection without blocking writes.
- **Write contention errors**: Increase `busy_timeout`. If you see `SQLITE_BUSY` in logs, the default 5 seconds isn't enough for your write volume.
- **Slow queries on large databases**: Increase `cache_size` (e.g., `-64000` for ~64 MB) and enable `mmap_size` (e.g., `268435456` for 256 MB). This keeps hot pages in memory.
- **Small VPS with limited RAM**: Keep defaults. The ~2 MB per-connection cache is intentionally conservative.

> **Note:** `cache_size` is per-connection. Total cache memory is roughly `cache_size × max_open_conns`. With the defaults (`-2000` and `2`), that's ~4 MB total.

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

### TCP Backlog

For servers handling many simultaneous connections:

```bash
sysctl -w net.core.somaxconn=4096
sysctl -w net.ipv4.tcp_max_syn_backlog=4096
```

Add to `/etc/sysctl.conf` to persist across reboots.

---

## Reverse Proxy Tuning

See the [Self-Hosting Guide](self-hosting.md#advanced-reverse-proxy) for nginx and Caddy configuration examples. The key considerations for scaling:

- **Disable response buffering** for SSE endpoints (`proxy_buffering off` in nginx, `flush_interval -1` in Caddy)
- **Set long read timeouts** on SSE paths (e.g., `proxy_read_timeout 86400s`) to prevent the proxy from killing idle SSE connections
- **Increase worker connections** in nginx (`worker_connections 4096`) if you expect thousands of concurrent SSE clients

---

## Example Configs

### Small (2 GB / 1 vCPU, ~100 users)

Use the defaults. No config changes needed.

### Medium (8 GB / 4 vCPU, ~1,000 users)

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

### Large (32 GB / 8 vCPU, ~10,000 users)

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
```

Also set `LimitNOFILE=65536` in the systemd unit for this profile.

---

## Monitoring

Key metrics to watch when scaling:

- **SQLite busy retries**: `SQLITE_BUSY` errors in logs indicate write contention. Increase `busy_timeout` or reduce `max_open_conns`.
- **SSE connection count**: Monitor the number of active SSE clients. Each consumes memory proportional to `client_buffer_size`.
- **Memory usage**: (`cache_size` x `max_open_conns`) + `mmap_size` + (SSE clients x buffer size x avg event size) gives a rough memory floor.
- **File descriptors**: `ls /proc/$(pidof enzyme)/fd | wc -l` shows current usage. Compare to `LimitNOFILE`.
- **Response latency**: If P99 response times degrade, the database may benefit from a larger cache or mmap.
