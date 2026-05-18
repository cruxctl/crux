# Crux CLI Architecture

`crux` is the operator CLI for Crux Control. It keeps user interaction local and delegates only daemon health/config/event operations to `cruxd`.

```text
crux
  CLI command routing
  context config
  typed daemon HTTP client
  YAML agent specs
  local agent state store
  PTY factory and runner
  PTY output normalization
  structured logging with rotation
  crux/cruxd update installer
```

The package boundaries are:

| Package | Responsibility |
|---|---|
| `configs/agents` | Embedded YAML definitions for known coding agents. |
| `internal/agent` | Agent spec loading, binary discovery, probe metadata, parser hooks, and filesystem state under `~/.crux/state`. |
| `internal/pty` | PTY factory, terminal wrapper, runner, matcher, recorder inputs, and output normalizer. |
| `internal/cli` | Command routing, output formatting, update flow, daemon-backed commands, and orchestration of agent PTY tasks. |
| `internal/client` | Small HTTP client for daemon health, version, runtime config, executions, and events. |
| `internal/config` | CLI contexts and path helpers. |
| `internal/logging` | Structured CLI logging with file rotation. |

Agent-specific behavior belongs in YAML. Go code stays generic: resolve a spec, build a PTY task, wait for readiness, send a command, normalize output, store raw/clean files, and print table/json/yaml output.
