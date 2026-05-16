# Architecture

The MVP is deliberately a single Go module with two binaries.

```text
crux
  CLI, context config, output formatting, HTTP client

cruxd
  HTTP API
  runtime config manager
  local JSON resource store
  managed CLI discovery
  worker limiter
  command runner
  event collector
```

The package boundaries are:

| Package | Responsibility |
|---|---|
| `internal/domain` | Stable resource and runtime types. |
| `internal/config` | YAML/env/flag configuration and CLI contexts. |
| `internal/store` | Store interface plus local JSON implementation. |
| `internal/discovery` | Managed CLI discovery adapters. |
| `internal/runner` | Command execution boundary. |
| `internal/worker` | Runtime concurrency primitives. |
| `internal/service` | Use-case orchestration. |
| `internal/api` | HTTP transport only. |
| `internal/client` | Typed API client for the CLI. |
| `internal/cli` | Operator command surface. |
| `internal/daemon` | Process wiring and graceful shutdown. |

The service layer depends on interfaces, not concrete transports. Future SQLite/Postgres stores, OpenTelemetry exporters, and runtime adapters should replace package internals without changing CLI command handlers or HTTP handlers.

