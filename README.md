# Crux

`crux` is the operator CLI for the Crux Control MVP. It talks to the daemon/server provided by [`cruxctl/cruxd`](https://github.com/cruxctl/cruxd).

The CLI repository intentionally contains only the user-facing command, context config, API client, output formatting, and daemon bootstrap flow.

## Scope

V0.1 CLI responsibilities:

- manage CLI contexts;
- call the `cruxd` HTTP API;
- register and inspect command-backed agents;
- discover managed CLI agents through `cruxd`;
- submit executions and read traces/events;
- update runtime config through `cruxd`;
- bootstrap daemon installation from the `cruxd` repo when `crux up` cannot find a running or installed daemon.

Deferred: Kubernetes, Docker Compose, agentgateway, MCP proxying, model routing, OIDC, console UI, approvals, AgBOM, and SDK adapters.

## Install

### From Source

Requirements:

- Go 1.26.2 or newer
- `make`

```bash
git clone https://github.com/cruxctl/crux.git
cd crux
make build
install -Dm755 bin/crux ~/.local/bin/crux
```

### Go Install

```bash
go install github.com/cruxctl/crux/cmd/crux@latest
```

Make sure `$(go env GOPATH)/bin` is on `PATH`.

## Quick Start

Run `crux up`. If `cruxd` is already online, it exits successfully. If `cruxd` is installed but offline, it starts `cruxd` in the foreground. If `cruxd` is missing, it prompts before downloading and running the daemon install script from the `cruxd` repo.

```bash
crux up
```

Non-interactive bootstrap:

```bash
crux up --yes
```

Use a custom install script URL for testing or pinned channels:

```bash
crux up --yes --install-script-url https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh
```

After the daemon is running:

```bash
crux doctor
crux agents add echo --cmd /usr/bin/printf --arg '%s\n' --arg '{prompt}'
crux run echo "hello from crux"
crux ps
crux trace last
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

Every command supports command-local help:

```bash
crux discover --help
crux agents describe --help
crux help config set
```

### Lifecycle

Ensure the daemon exists and is running:

```bash
crux up
crux up --yes
crux up --no-start --yes
crux up --install-script-url https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh
```

When a local `cruxd` binary is already installed but offline, `crux up` starts it in the foreground. These flags are passed through to `cruxd`:

```bash
crux up --daemon-config ~/.config/crux/cruxd.yaml
crux up --address 127.0.0.1 --port 7790
crux up --store /tmp/crux-state.json
crux up --api-key local-dev-key
```

Check health and versions:

```bash
crux doctor
crux version
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

Update daemon tunables through the API:

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

Register a command-backed agent. If any argument contains `{prompt}`, `cruxd` replaces it with the run prompt; otherwise the prompt is sent to stdin.

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

Discover installed managed CLI agents on the daemon host:

```bash
crux discover
crux -o json discover
```

Current discovery candidates are `claude`, `codex`, `gemini`, and `kimi`. The daemon searches its service `PATH` plus common user binary locations, including NVM-managed Node.js bins.

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
crux events ls
crux -o json events
```

## Daemon Install Script

The default bootstrap script URL is:

```text
https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh
```

The script is owned by `cruxctl/cruxd`, not this CLI repo.

## Development

```bash
make fmt
make lint
make test
make build
```
