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

## Install

### From source

Requirements:

- Go 1.26.2 or newer
- `make`

Build both binaries locally:

```bash
git clone https://github.com/cruxctl/crux.git
cd crux
make build
```

The binaries are written to:

```text
bin/crux
bin/cruxd
```

Install them into a directory on `PATH`:

```bash
install -Dm755 bin/crux ~/.local/bin/crux
install -Dm755 bin/cruxd ~/.local/bin/cruxd
```

Or run without installing:

```bash
./bin/crux --help
./bin/cruxd --help
```

### Go install

For a direct Go install:

```bash
go install github.com/cruxctl/crux/cmd/crux@latest
go install github.com/cruxctl/crux/cmd/cruxd@latest
```

This installs into `$(go env GOPATH)/bin`. Make sure that directory is on `PATH`.

## Quick Start

Start the daemon:

```bash
cruxd --config examples/cruxd.yaml
```

Or start it through the CLI:

```bash
crux up --daemon-config examples/cruxd.yaml
```

In another terminal:

```bash
crux doctor
crux agents add echo --cmd /usr/bin/printf --arg '%s\n' --arg '{prompt}'
crux run echo "hello from crux"
crux ps
crux trace last
```

Discover installed managed CLIs:

```bash
crux discover
```

Update runtime config without rebuilding or restarting:

```bash
crux config set --concurrency 4 --job-timeout 1200 --max-output-bytes 2097152
crux -o yaml config get
```

## Usage

Global form:

```bash
crux [global flags] <command> [args]
```

Global flags:

| Flag | Purpose |
|---|---|
| `--config PATH` | CLI config file. Default: `~/.config/crux/config.yaml`. |
| `--context NAME` | Use a named CLI context for one command. |
| `--server URL` | Override the active context's `cruxd` URL. |
| `--api-key KEY` | Override the active context's API key. |
| `-o, --output FMT` | Output format: `table`, `json`, or `yaml`. |

Global flags must appear before the command:

```bash
crux -o yaml config get
crux --server http://127.0.0.1:7790 doctor
```

### Lifecycle

Run `cruxd` in the foreground:

```bash
crux up
crux up --daemon-config examples/cruxd.yaml
crux up --address 127.0.0.1 --port 7790 --store /tmp/crux-state.json
crux up --api-key local-dev-key
```

Check health and versions:

```bash
crux doctor
crux version
```

The daemon binary accepts the same local service flags:

```bash
cruxd
cruxd --config examples/cruxd.yaml
cruxd --address 127.0.0.1 --port 7790
cruxd --store /tmp/crux-state.json
cruxd --api-key local-dev-key
```

### Contexts

Contexts store server URLs and optional API keys for the CLI:

```bash
crux context ls
crux context current
crux context set local --server http://127.0.0.1:7700 --namespace default
crux context set secure-local --server http://127.0.0.1:7700 --api-key local-dev-key
crux context use secure-local
```

### Runtime Config

Read the active runtime config:

```bash
crux config get
crux -o json config get
crux -o yaml config get
```

Update tunables through the API:

```bash
crux config set --concurrency 4
crux config set --job-timeout 1200
crux config set --max-output-bytes 2097152
crux config set --discovery-timeout 5
crux config set --trace-retention 20000
crux config set --log-level debug
crux config set --namespace platform
crux config set --allow-shell=true
```

These flags can be combined:

```bash
crux config set --concurrency 4 --job-timeout 1200 --max-output-bytes 2097152
```

### Agents

List and inspect registered agents:

```bash
crux agents ls
crux agents describe echo
crux -o yaml agents describe echo
```

Register a command-backed agent. If any argument contains `{prompt}`, Crux replaces it with the run prompt; otherwise the prompt is sent to stdin.

```bash
crux agents add echo --cmd /usr/bin/printf --arg '%s\n' --arg '{prompt}'
crux agents add cat --cmd /usr/bin/cat
crux agents add worker --cmd /usr/local/bin/worker --arg run --arg '{prompt}' --workdir /tmp
crux agents add worker --cmd /usr/local/bin/worker --env FOO=bar --timeout 300
```

Remove an agent:

```bash
crux agents rm echo
```

Discover installed managed CLI agents on `PATH`:

```bash
crux discover
crux -o json discover
```

Current discovery candidates are `claude`, `codex`, `gemini`, and `kimi`.

### Executions

Run an agent and wait for output:

```bash
crux run echo "hello from crux"
```

Queue an execution asynchronously:

```bash
crux run echo "background job" --async
```

List executions:

```bash
crux ps
crux -o json ps
```

Show execution events:

```bash
crux trace last
crux trace <execution-id>
crux -o yaml trace <execution-id>
```

Show all daemon events:

```bash
crux events
crux -o json events
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
