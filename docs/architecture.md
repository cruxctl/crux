# Crux CLI Architecture

`crux` is the operator CLI for the Crux MVP. The daemon/server lives in the separate [`cruxctl/cruxd`](https://github.com/cruxctl/cruxd) repository.

```text
crux
  CLI command routing
  context config
  output formatting
  typed HTTP client
  managed-agent operation commands
  rotated CLI logging
  crux/cruxd update installer
```

The package boundaries are:

| Package | Responsibility |
|---|---|
| `internal/config` | CLI contexts and path helpers. |
| `internal/client` | Typed HTTP client for `cruxd`. |
| `internal/logging` | Structured CLI logging with file rotation. |
| `internal/cli` | Operator command surface, managed-agent usage/cost/external-session/history/fallback workflows, and `crux update` install/update flow. |

The CLI imports only `github.com/cruxctl/cruxd/pkg/cruxapi` from the daemon repo for request/response types. It does not embed the daemon, store, worker, runner, or HTTP server.
