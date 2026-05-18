# Crux

`crux` is the operator CLI for Crux Control. It owns the local user-facing command surface, YAML-driven coding-agent discovery, PTY automation, output normalization, context config, output formatting, structured logging, and the installer/update flow.

The daemon/server lives in [`cruxctl/cruxd`](https://github.com/cruxctl/cruxd). The CLI still uses `cruxd` for daemon health, runtime config, and event tracing, while coding-agent discovery and PTY execution now run locally through the CLI.

## Scope

Current CLI responsibilities:

- manage CLI contexts;
- call `cruxd` health, version, runtime config, and event APIs;
- discover coding-agent CLIs from built-in YAML specs;
- store discovered-agent state under `~/.crux/state`;
- run agent probes and interactive sessions through a reusable PTY factory;
- normalize raw PTY output before parsing;
- support `table`, `json`, and `yaml` output on every command;
- install or update both `crux` and `cruxd` through explicit installer scripts.

## Install

### Linux and macOS

```bash
curl -fsSL https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.sh | sh
```

Force-refresh the local daemon binary and user service:

```bash
curl -fsSL https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.sh | sh -s -- --force
```

### Windows

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

## Quick Start

```bash
crux update --yes
crux doctor
crux discover
crux ps
crux claude-code describe
crux gemini-cli usage
crux aider exec --repo .
crux opencode conversations ls
```

## Usage

Global form:

```bash
crux [global flags] <command> [args]
crux <command> [args] [global flags]
crux <agent-name> <describe|usage|exec|conversations>
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

Every command supports `-h` and `--help`.

## Agent Commands

Agent behavior is defined in `configs/agents/*.yaml`. Current built-in specs cover:

- `claude-code` (`claude`)
- `gemini-cli` (`gemini`)
- `aider`
- `codex`
- `opencode`
- `kimi-cli` (`kimi`)
- `cline` as config/file discovery only

Discover installed agents and save state:

```bash
crux discover
crux discover -o json
```

List discovered agents:

```bash
crux ps
crux ps -o yaml
```

Inspect one agent:

```bash
crux claude-code describe
crux gemini-cli describe -o json
```

Collect usage or stats through a YAML-defined PTY probe:

```bash
crux gemini-cli usage
crux aider usage
```

List known conversations or sessions when the agent exposes a TUI command:

```bash
crux opencode conversations ls
crux kimi-cli conversations ls
```

Open an interactive managed PTY session:

```bash
crux claude-code exec
crux gemini-cli exec --repo .
crux codex exec -- --no-alt-screen
```

## PTY Output Pipeline

Crux always stores raw PTY output and parses only normalized text:

```text
PTY raw bytes
  -> raw recorder
  -> ANSI/control cleanup
  -> box/spinner/redraw cleanup
  -> clean text and final screen
  -> parser hooks
  -> ~/.crux/state
```

Probe/session files are written below:

```text
~/.crux/state/
  agents/<agent>/
  sessions/<session>/
```

## Daemon Commands

The daemon-backed commands remain:

```bash
crux doctor
crux version
crux context ls
crux config get
crux config set --concurrency 4
crux trace last
crux events
```

## Development

```bash
make fmt
make lint
make test
make build
```
