# Crux

Crux is the simplified Go MVP for a vendor-neutral AI agent control plane. It intentionally starts with two binaries:

- `crux` - operator CLI.
- `cruxd` - local control-plane daemon that combines server, worker, gateway boundary, and event collector in one process.

This repo is the Cruxctl-owned MVP and does not depend on the `github-agenticfleet/crux-*` repositories.

## Scope

V0.1 proves the smallest useful loop:

- read config from YAML, environment variables, and command-line flags;
- persist local state in a JSON store;
- register command-backed agents;
- discover common managed CLI agents on `PATH`;
- run agents as subprocesses with prompt injection through `{prompt}` args or stdin;
- capture execution status, stdout, stderr, and append-only events;
- update runtime config through `PATCH /v1/config/runtime`;
- operate everything through the `crux` CLI.

Deferred: Kubernetes, Docker Compose, agentgateway, MCP proxying, model routing, OIDC, console UI, approvals, AgBOM, and SDK adapters.

## Build

```bash
make fmt
make test
make build
```

## Run

Start the daemon:

```bash
./bin/cruxd --config examples/cruxd.yaml
```

Or start it through the CLI:

```bash
./bin/crux up --daemon-config examples/cruxd.yaml
```

In another terminal:

```bash
./bin/crux doctor
./bin/crux agents add echo --cmd /usr/bin/printf --arg '%s\n' --arg '{prompt}'
./bin/crux run echo "hello from crux"
./bin/crux ps
./bin/crux trace last
```

Discover installed managed CLIs:

```bash
./bin/crux discover
```

Update runtime config without rebuilding or restarting:

```bash
./bin/crux config set --concurrency 4 --job-timeout 1200 --max-output-bytes 2097152
./bin/crux -o yaml config get
```

## Configuration

Precedence for daemon settings is:

1. CLI flags
2. environment variables
3. YAML config
4. defaults

Default daemon config path: `~/.config/crux/cruxd.yaml`.

Default CLI context path: `~/.config/crux/config.yaml`.

Important daemon environment variables:

| Variable | Purpose |
|---|---|
| `CRUX_SERVER_ADDRESS` | HTTP bind address. |
| `CRUX_SERVER_PORT` | HTTP bind port. |
| `CRUX_STORE_PATH` | JSON state path. |
| `CRUX_API_KEY` | Optional local API key. |
| `CRUX_WORKER_CONCURRENCY` | Max simultaneous command executions. |
| `CRUX_JOB_TIMEOUT_SECONDS` | Default command timeout. |
| `CRUX_MAX_OUTPUT_BYTES` | Max captured stdout and stderr bytes per stream. |
| `CRUX_DISCOVERY_TIMEOUT_SECONDS` | Per-command discovery timeout. |
| `CRUX_TRACE_RETENTION_ENTRIES` | Max retained append-only events. |

CLI environment overrides:

| Variable | Purpose |
|---|---|
| `CRUX_SERVER_URL` | Overrides active context server URL. |
| `CRUX_API_KEY` | Overrides active context API key. |

## API

The daemon exposes:

```text
GET    /healthz
GET    /v1/version
GET    /v1/config
PATCH  /v1/config/runtime
GET    /v1/agents
POST   /v1/agents
GET    /v1/agents/{name}
DELETE /v1/agents/{name}
POST   /v1/discover
GET    /v1/executions
POST   /v1/executions
GET    /v1/executions/{id}
GET    /v1/executions/{id}/events
GET    /v1/events
```
