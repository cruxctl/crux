# Crux

`crux` is the operator CLI for the Crux Control MVP. It talks to the daemon/server provided by [`cruxctl/cruxd`](https://github.com/cruxctl/cruxd).

The CLI repository intentionally contains only the user-facing command, context config, API client, output formatting, logging, and installer/update flow.

## Scope

V0.1 CLI responsibilities:

- manage CLI contexts;
- call the `cruxd` HTTP API;
- register and inspect command-backed agents;
- discover managed CLI agents through `cruxd`;
- submit transcript-backed TTY executions and read traces/events;
- read unified managed-agent usage, cost signals, sessions, Crux execution history, resume plans, and fallback settings from imported TTY transcripts;
- open managed-agent TUIs through daemon-planned commands, capture PTY transcripts with `expect` or `script`, and import those transcripts into Crux history;
- update runtime config through `cruxd`;
- install or update both `crux` and `cruxd` through explicit installer scripts.

Deferred: Kubernetes, Docker Compose, agentgateway, MCP proxying, model routing, OIDC, console UI, approvals, AgBOM, and SDK adapters.

## Install

### Linux and macOS

The all-in-one installer installs `cruxd` first, then installs the `crux` CLI:

```bash
curl -fsSL https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.sh | sh
```

Force-refresh the local daemon binary and user service:

```bash
curl -fsSL https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.sh | sh -s -- --force
```

By default the CLI installer resolves the `cruxd` installer to a commit-pinned raw GitHub URL before running it, so branch-level raw cache cannot install a stale daemon. Set `CRUXD_INSTALL_REF` to test a different daemon branch, tag, or commit.

Delete the local CLI binary without removing daemon state:

```bash
rm -f ~/.local/bin/crux
```

Delete the CLI and stop/remove the user daemon service:

```bash
rm -f ~/.local/bin/crux
systemctl --user disable --now cruxd.service
rm -f ~/.config/systemd/user/cruxd.service ~/.local/bin/cruxd
systemctl --user daemon-reload
```

### Windows

PowerShell installs the current-user binaries and registers `cruxd` as a scheduled task:

```powershell
iwr https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.ps1 -UseB | iex
```

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

Run `crux update` to install or update both the CLI and daemon. Use `--force` when you want to purge and reinstall the local `cruxd` binary and user service.

```bash
crux update
crux update --force
```

Non-interactive update:

```bash
crux update --yes
```

Update only one component:

```bash
crux update --component crux
crux update --component cruxd --force
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
crux <command> [args] [global flags]
```

Global flags:

| Flag | Purpose |
|---|---|
| `--config PATH` | CLI config file. Default: `~/.config/crux/config.yaml`. |
| `--context NAME` | Use a named CLI context for one command. |
| `--server URL` | Override the active context's `cruxd` URL. |
| `--api-key KEY` | Override the active context's API key. |
| `-o, --output FMT` | Output format: `table`, `json`, or `yaml`. |
| `--log-level LEVEL` | CLI log level: `debug`, `info`, `warn`, or `error`. |
| `--log-file PATH` | Rotated CLI log file path. Use `none` to disable file logging. |

Global flags may appear before the command or after command arguments:

```bash
crux -o yaml config get
crux config get -o yaml
crux agent gemini usage -o json
crux --server http://127.0.0.1:7790 doctor
```

Every command supports command-local help:

```bash
crux discover --help
crux agents describe --help
crux help config set
```

### Lifecycle

Install or update components:

```bash
crux update
crux update --component crux
crux update --component cruxd
crux update --component cruxd --force
crux update --version latest --yes
crux update --no-start
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
crux agents
crux agents describe echo
crux agents describe echo -o yaml
crux agent echo
crux agent echo describe
crux agent claude usage
crux agent claude cost
crux agent claude sessions
crux agent gemini sessions
crux agent codex sessions -o json
crux agent codex fallback set gemini,claude
crux agent claude usage -o json
```

`crux run` now uses the same TTY exec plan/record path as `crux agent <name> exec`. The CLI asks `cruxd` for a provider-specific TUI command, opens it locally, inserts the prompt, captures the transcript, and imports the result as an immutable Crux execution. The current working directory is sent with the exec plan unless the registered agent has an explicit `--workdir`.

Managed-agent session, usage, cost, and history views are transcript-backed. Provider-private session stores are not read directly by `cruxd`; use `crux agent <name> exec`, `crux run`, `crux agent <name> resume`, or history sharing to import provider TTY activity into Crux.

Open a managed provider TUI through Crux:

```bash
crux agent codex exec
crux agent claude exec "inserting into this session"
crux agent codex exec --resume last -- --no-alt-screen
crux agent gemini exec --send "/help" --send "/quit" --expect "help" --timeout 30
crux agent claude exec --dry-run
crux agent kimi exec --transcript ~/.local/state/crux/tty/kimi-session.log
```

`crux agent <name> exec "<text>"` asks `cruxd` for the provider-specific TUI command, then runs it locally with `expect` when available, otherwise `script`, otherwise a direct subprocess fallback. Transcripts are stored under `~/.local/state/crux/tty/` by default and posted back to `cruxd` when the session exits. That import creates a normal immutable Crux execution, so `crux ps`, `crux trace`, `crux agent <name> usage`, `crux agent <name> cost`, `crux agent <name> sessions`, and history share/replay commands all read the same transcript-backed state.

Use `--send` and `--expect` for scripted probes. Include the provider's normal quit command in `--send` when automating a TUI, and set `--timeout` to cap the scripted session. Use `--no-record` only when you want a local transcript without adding it to Crux history.

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
crux agent echo rm
```

Fleet monitoring:

```bash
crux agents usage
crux agents cost
crux agents sessions
```

Discover installed managed CLI agents on the daemon host:

```bash
crux discover
crux -o json discover
```

Current discovery candidates are `claude`, `codex`, `gemini`, and `kimi`. The daemon searches its service `PATH` plus common user binary locations, including NVM-managed Node.js bins. Discovery keeps the registered command paths for identity and custom-agent compatibility; managed-agent run/resume/exec behavior is planned through the TTY adapter layer.

### Executions

Run an agent and wait for output:

```bash
crux run echo "hello from crux"
crux run echo "hello from crux" -o yaml
crux run codex "explain this repo" --fallback gemini,claude
crux run gemini --resume latest "continue with a shorter answer"
crux run codex --resume 019e3715-cdcc-78a2-bc76-a40edfb67fdc "continue this outside-Crux session"
crux run claude --from exec_abc123 --prompt "turn the previous answer into a checklist"
```

`--async` is retired for `crux run` because runs now depend on a local TTY transcript and must stay attached until the transcript is recorded.

List executions:

```bash
crux ps
crux ps --agent gemini
crux ps --status failed --last 5
crux -o json ps
```

View and share immutable Crux history:

```bash
crux agent gemini history
crux agent gemini history show exec_abc123
crux agent gemini history share exec_abc123 codex --prompt "review and improve this"
crux agent claude resume last "continue the previous Claude session"
crux agent kimi resume 567cbc06-5586-4110-bb57-6cb6c16ce761 "continue this Kimi session from Crux"
```

Cost views are evidence-based. Crux shows realtime local execution state, duration, output volume, and provider usage text only when that evidence appears in imported TTY transcripts.

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

## Install Scripts

The default installer URLs are:

```text
https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.sh
https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.ps1
https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh
https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.ps1
```

The `crux` installer calls the `cruxd` installer first unless `--skip-cruxd` is passed. Preview installers build native binaries with Go for Linux, macOS, and Windows on amd64 or arm64 hosts. When the default daemon installer URL is used, the wrapper resolves `CRUXD_INSTALL_REF` through GitHub and downloads the script from the resolved commit SHA instead of a mutable branch URL.

## Development

```bash
make fmt
make lint
make test
make build
```
