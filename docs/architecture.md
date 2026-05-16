# Crux CLI Architecture

`crux` is the operator CLI for the Crux MVP. The daemon/server lives in the separate [`cruxctl/cruxd`](https://github.com/cruxctl/cruxd) repository.

```text
crux
  CLI command routing
  context config
  output formatting
  typed HTTP client
  cruxd bootstrap installer
```

The package boundaries are:

| Package | Responsibility |
|---|---|
| `internal/config` | CLI contexts and path helpers. |
| `internal/client` | Typed HTTP client for `cruxd`. |
| `internal/cli` | Operator command surface and `crux up` bootstrap flow. |

The CLI imports only `github.com/cruxctl/cruxd/pkg/cruxapi` from the daemon repo for request/response types. It does not embed the daemon, store, worker, runner, or HTTP server.
